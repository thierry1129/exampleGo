package controllers
import (
	"github.com/astaxie/beego"
	"github.com/bndr/gojenkins"
	"fmt"
	"encoding/json"
	"strings"
	"strconv"
	"garden/models/rest"
	"github.com/beevik/etree"
	"errors"
	//"time"
	//"net/url"
	//"github.com/go-openapi/spec"
	//"time"
)

var (
	jenkins *gojenkins.Jenkins
)

type JenkinsController struct {
	beego.Controller
}


//@Title: Post
//@Description: post a new job to jenkins
//@author: 吕帅林
//@router / [post]
func (j *JenkinsController) Post(){
	Init()
	var job rest.PostJob
	json.Unmarshal(j.Ctx.Input.RequestBody, &job)

	var configString string
	configString += `<?xml version="1.0" encoding="UTF-8"?>
<maven2-moduleset plugin="maven-plugin@2.16">
	<actions></actions>
	<description>`+job.Description+`</description>
	<keepDependencies>false</keepDependencies>
	<properties>`

	//build discarder property of jenkins config.
	if job.DiscardOld {
		configString+=`<jenkins.model.BuildDiscarderProperty>
			<strategy class="hudson.tasks.LogRotator">
				<daysToKeep>`+strconv.Itoa(job.DaysToKeep)+`</daysToKeep>
				<numToKeep>`+strconv.Itoa(job.NumToKeep)+`</numToKeep>
				<artifactDaysToKeep>-1</artifactDaysToKeep>
				<artifactNumToKeep>-1</artifactNumToKeep>
			</strategy>
		</jenkins.model.BuildDiscarderProperty>`
	}
	configString+=`</properties>
	<scm class="hudson.plugins.git.GitSCM" plugin="git@3.3.0">
		<configVersion>2</configVersion>
		<userRemoteConfigs>
			<hudson.plugins.git.UserRemoteConfig>
				<url>`+job.GitUrl+`</url>
				<credentialsId>`+job.GitAuth+`</credentialsId>
			</hudson.plugins.git.UserRemoteConfig>
		</userRemoteConfigs>
		<branches>`
// 循环增加 git branch 名字
		for j:=0 ; j<len(job.BranchName) ; j++{
			configString+=`<hudson.plugins.git.BranchSpec>
				<name>`+job.BranchName[j]+`</name>
			</hudson.plugins.git.BranchSpec>`
		}
		configString+=`</branches>
		<doGenerateSubmoduleConfigurations>false</doGenerateSubmoduleConfigurations>
		<submoduleCfg class="list"/>
		<extensions/>
	</scm>
	<canRoam>true</canRoam>
	<disabled>false</disabled>
	<blockBuildWhenDownstreamBuilding>false</blockBuildWhenDownstreamBuilding>
	<blockBuildWhenUpstreamBuilding>false</blockBuildWhenUpstreamBuilding>
	<triggers>
		<hudson.triggers.SCMTrigger>
			<spec>`+job.TriggerSpec+`</spec>
			<ignorePostCommitHooks>`+BoolToString(job.IgnorePostCommitHooks)+`</ignorePostCommitHooks>
		</hudson.triggers.SCMTrigger>
	</triggers>
	<concurrentBuild>false</concurrentBuild>`

	//如果job 有Discarder property, 则在config xml中加入相关字段
	if job.RootPom != "pom.xml" {
		configString+="<rootPOM>"+job.RootPom+"</rootPOM>"
	}
	if job.PomGoal != "" {
		configString+="<goals>"+job.PomGoal+"</goals>"
	}
	 configString += `<aggregatorStyleBuild>true</aggregatorStyleBuild>
	<incrementalBuild>false</incrementalBuild>
	<ignoreUpstremChanges>false</ignoreUpstremChanges>
	<ignoreUnsuccessfulUpstreams>false</ignoreUnsuccessfulUpstreams>
	<archivingDisabled>false</archivingDisabled>
	<siteArchivingDisabled>false</siteArchivingDisabled>
	<fingerprintingDisabled>false</fingerprintingDisabled>
	<resolveDependencies>false</resolveDependencies>
	<processPlugins>false</processPlugins>
	<mavenValidationLevel>0</mavenValidationLevel>
	<runHeadless>false</runHeadless>
	<disableTriggerDownstreamProjects>false</disableTriggerDownstreamProjects>
	<blockTriggerWhenBuilding>false</blockTriggerWhenBuilding>
	<settings class="jenkins.mvn.DefaultSettingsProvider"/>
	<globalSettings class="jenkins.mvn.DefaultGlobalSettingsProvider"/>
	 <reporters>
	 <hudson.maven.reporters.MavenMailer>
      <recipients>`+job.Recipients+`</recipients>
      <dontNotifyEveryUnstableBuild>false</dontNotifyEveryUnstableBuild>
      <sendToIndividuals>false</sendToIndividuals>
      <perModuleEmail>true</perModuleEmail>
    </hudson.maven.reporters.MavenMailer>
  	</reporters>
	<publishers/>
	<buildWrappers/>
	<prebuilders/>
	<postbuilders>
		<hudson.tasks.Shell>
			<command>`+ProcessLocalCommand(job.Command)+`</command>
		</hudson.tasks.Shell>
		<org.jvnet.hudson.plugins.SSHBuilder plugin="ssh@2.4">
      		<siteName>`+job.SiteName+`</siteName>
      		<command>`+job.SshCommand+`</command>
    	</org.jvnet.hudson.plugins.SSHBuilder>
	</postbuilders>
	<runPostStepsIfResult>
		<name></name>
		<ordinal></ordinal>
		<color></color>
		<completeBuild></completeBuild>
	</runPostStepsIfResult>
</maven2-moduleset>`

	_,errCreate := jenkins.CreateJob(configString, job.Name)
	if errCreate != nil {
		fmt.Println("below is the error to create new job ")
		fmt.Println(errCreate)
		//如果 创建失败 则发送 失败的讯息
		data := make(map[string]string)
		data["Error"] = errors.New("Create job failed").Error()
		j.Data["json"] = data
	} else {
		//如果创建任务成功，则发送名字
		j.Data["json"] = map[string]string{"name": job.Name}
	}
	j.ServeJSON()
}

//main函数为测试专用
func main() {
}


//@Title: GetAllJobs
//@Description: get all jobs and then send as a json string
//@author: 吕帅林
//@email: terrylu@wustl.edu
//@router / [get]
func (j *JenkinsController) GetAllJobs (){
	Init()
	var formattedJobList = GetAllJobsHelper()
	j.Data["json"] = formattedJobList
	j.ServeJSON()
}


//@Title: Get
//@Description: get a specific job from jenkins using job id
//@router /getJob/:jid [get]
func (j *JenkinsController) Get (){
	Init()
	jobName := j.GetString(":jid")
	var jobList []string

	if	jobName != "" {
		job,err := jenkins.GetJob(jobName)
		if err != nil{
			beego.BeeLogger.Error("there is no job with such name")
		}
		if job!=nil {
			var jobDetailed rest.PostJob
			jobDetailed.Name = job.GetName()
			var jobConfig, err2= job.GetConfig()
			doc := etree.NewDocument()
			doc.ReadFromString(jobConfig)
			var root= doc.Root()
			var branchNameArr []string
			var buildDiscard = false
			var daysToKeep int
			var numToKeep int
			var pomGoal string = ""
			var rootPom string = "pom.xml"
			//var oneJob rest.PostJob
			if err2 != nil {
				beego.BeeLogger.Error("Error! getting job config error")
			}
				properties := root.SelectElement("properties")
				buildDiscarder := properties.SelectElement("jenkins.model.BuildDiscarderProperty")
				scm := root.SelectElement("scm")
				userRemoteConfig := scm.SelectElement("userRemoteConfigs")
				hudson1 := userRemoteConfig.SelectElement("hudson.plugins.git.UserRemoteConfig")
				url := hudson1.SelectElement("url")
				credentialsId := hudson1.SelectElement("credentialsId")
				branches := scm.SelectElement("branches")
				hudson2 := branches.SelectElements("hudson.plugins.git.BranchSpec")



			for _,childElement := range hudson2 {
				branchNameArr = append(branchNameArr,childElement.SelectElement("name").Text())
			}

				triggers := root.SelectElement("triggers")
				hudson3 := triggers.SelectElement("hudson.triggers.SCMTrigger")
				spec := hudson3.SelectElement("spec")
				pom := root.SelectElement("rootPOM")
				goals := root.SelectElement("goals")
				reporters := root.SelectElement("reporters")
				hudson4 := reporters.SelectElement("hudson.maven.reporters.MavenMailer")
				postbuilders := root.SelectElement("postbuilders")
				hudson5 := postbuilders.SelectElement("hudson.tasks.Shell")
				var test = "org.jvnet.hudson.plugins.SSHBuilder"
				org := postbuilders.SelectElement(test)

				if pom!=nil{
				rootPom = pom.Text()
				}
				if goals!=nil{
				pomGoal = goals.Text()
				}

				//如果用户勾选了 丢弃旧的构建按钮
				if buildDiscarder != nil {
					buildDiscard = true
					strat := buildDiscarder.SelectElement("strategy")
					daysToKeep, _ = strconv.Atoi(strat.FindElement("daysToKeep").Text())
					numToKeep, _ = strconv.Atoi(strat.SelectElement("numToKeep").Text())
				}
			fmt.Println("in get method, the description acquired is "+job.GetDescription())

					oneJob := &rest.PostJob{
						Name:        job.GetName(),
						Description: job.GetDescription(),
						DaysToKeep:  daysToKeep,
						NumToKeep:  numToKeep,
						GitUrl:     url.Text(),
						DiscardOld: buildDiscard,
						PomGoal: pomGoal,
						RootPom: rootPom,
						GitAuth:               credentialsId.Text(),
						BranchName:            branchNameArr,
						TriggerSpec:           spec.Text(),
						IgnorePostCommitHooks: StringToBool(hudson3.SelectElement("ignorePostCommitHooks").Text()),
						Recipients:            hudson4.SelectElement("recipients").Text(),
						Command:               hudson5.SelectElement("command").Text(),
						SiteName:              org.SelectElement("siteName").Text(),
						SshCommand:            org.SelectElement("command").Text(),
					}
			jsonJob, _ := json.Marshal(oneJob)
			jobString := string(jsonJob)
			jobList = append(jobList, jobString)
			j.Data["json"] = jobList
			j.ServeJSON()

			}

		}else{
		beego.BeeLogger.Error("name is empty ")
		}
	}


//@Title: VerifySCM
//@Description: verify scm input
//@router /verifyscm/ [post]
func (j *JenkinsController) VerifySCM (){
	Init()
	var scm scmVerify
	fmt.Println("the request body is ")
	fmt.Println(string(j.Ctx.Input.RequestBody))
	json.Unmarshal(j.Ctx.Input.RequestBody, &scm)

	fmt.Println("job name is "+scm.JobName)
	fmt.Println("the command is "+scm.ScmCommand)
	var result = ScmHelper(scm.JobName,scm.ScmCommand)

	//	//如果 scmhelper的返回值的 第十二个字符是e （代表 error )则 发送 error讯息
		if string(result[11]) == "e" {
			fmt.Println("this should be an error ")
			j.Data["json"] = map[string]string{"error": "failure"}
		} else {
			j.Data["json"] = map[string]string{"success": "success"}
		}
		j.ServeJSON()
}

//@Title: BuildStatus
//@Description: get if last build is finished or not,
//@router /checkLastBuild [get]
func(j *JenkinsController) BuildStatus() {
	Init()
	statusList := make([]string, 0)
	var jobList, err= jenkins.GetAllJobs()
	if err != nil {
		beego.BeeLogger.Info("there is no job in the system")
	}
	for _, job := range jobList {
		var buildStatus buildStatus
		buildStatus.Name = job.Raw.Name
		var lastBuild, err1= job.GetLastBuild()
		if err1 != nil {
			buildStatus.Status = "neverbuild"
			//there is no last build available
		} else {
			if lastBuild.IsRunning() {

				buildStatus.Status = "building"
			} else if lastBuild.Raw.Result == "SUCCESS" {
				buildStatus.Status = "success"

			} else {
				buildStatus.Status = "failure"
			}

		}
		testJson, err := json.Marshal(&buildStatus)

		if err != nil {
			beego.BeeLogger.Critical("there is an error converting job to json format")
		}

		var stringJson = string(testJson)
		//fmt.Println(stringJson)
		statusList = append(statusList, stringJson)
	}

	j.Data["json"] = statusList
	j.ServeJSON()

}



//@Title: BuildJob
//@Description: Build a selected job from jenkins
//@router /build/:name [Get]
func(j *JenkinsController) BuildJobs(){
	Init()
	var name = j.GetString(":name")
	var result,err = jenkins.BuildJob(name)
	if err != nil{
		panic("build job error")
	}
	j.Data["json"] = map[string]string{"success": strconv.FormatInt(result,10)}
	fmt.Println("below is the json data ")
	fmt.Println(result)
	j.ServeJSON()
}

//@Title: GetJobLogs
//@Description: get job build details
//@router /:name/logs [Get]
func (j *JenkinsController) GetJobLogs() {
	Init()

	var name = j.GetString(":name")
	//var name = "mavenTestNoDelete"

	jobList := make([]string, 0)
	var job,err = jenkins.GetJob(name)
	if err != nil {
		panic(err)
	}
	var buildIdArr ,err2 = job.GetAllBuildIds()
	if err2 != nil {
		panic(err2)
	}

//从 api中取出的 buildid array 从最大的开始 而且有的时候 build 1 ,2,3..会缺失，所以使用
	//下面的循环方法
	for a := buildIdArr[len(buildIdArr)-1].Number ;a<=buildIdArr[0].Number;a++{
		var singleBuild,err3 = jenkins.GetBuild(name,int64(a))
		if err3!=nil{
			beego.BeeLogger.Error("there is an error retrieving build")
		}
			oneJob := &jobLog{
				BuildId: int(a),
				Success: singleBuild.IsGood(),
				TimeStamp: singleBuild.GetTimestamp().String(),

			}
			testJson,err := json.Marshal(&oneJob)
			if err != nil{
				panic("there is an error converting job log to json format")
			}
			var stringJson = string(testJson)
			fmt.Println(stringJson)
			jobList = append(jobList, stringJson)
	}

	j.Data["json"] = jobList
	j.ServeJSON()

}

//@Title: GetBuildDetails
//@Description: get build details from jenkins
//@router /:name/logs/:buildid [Get]
func (j *JenkinsController) GetBuildDetails(){
	Init()

	var name = j.GetString(":name")
	var buildId,err = j.GetInt64(":buildid")
	jobList := make([]string, 0)
	if err != nil {
		panic(err)
	}
	var job,err2 = jenkins.GetJob(name)
	if err2 != nil {
		panic(err)
	}
	var singleBuild,err3= job.GetBuild(buildId)

	if err3 != nil {
		panic(err3)
	}
	var singleLog = singleBuild.GetConsoleOutput()
	oneJob := &jobDetail{
		BuildId: int(buildId),
		Details: singleLog,
	}
	testJson,err := json.Marshal(&oneJob)
	if err != nil{
		beego.BeeLogger.Error("error while converting job to json format ")
	}
	var stringJson = string(testJson)
	jobList = append(jobList, stringJson)
	j.Data["json"] = jobList
	j.ServeJSON()
}


//@Title: DeleteJobs
//@Description: Delete a selected job from jenkins
//@router /:name [delete]
func (j *JenkinsController) DeleteJobs() {
	Init()
	//这个函数 同时处理删除一个任务 和 删除多个任务。 参数是以逗号分隔的字符串
	var name = j.GetString(":name")
	var nameArr = strings.FieldsFunc(name, Split)
	for _,jobName := range nameArr {
		fmt.Println(jobName)
		success,err := jenkins.DeleteJob(jobName)
		if err!=nil {
			panic(err)
		}
		fmt.Println(success)
	}
}


//@Title: VerifyEmail
//@Description: verify if the email is in correct format
//@router /verifyEmail/:email [get]
func (j *JenkinsController) VerifyEmail(){
	var email = j.GetString(":email")
	fmt.Print("the email got from the front page is "+email)
	if VerifyEmailHelper(email) {
		j.Data["json"] = map[string]string{"success": "success"}
		//j.Ctx.Output.Body([]byte("success"))
		fmt.Println("it is a success")
	}else {

		j.Data["json"] = map[string]string{"error": "failure"}
		//j.Ctx.Output.Body([]byte("error"))
		fmt.Println("there is an error")
	}
	j.ServeJSON()
}

//@Title: Put
//@Description: update a selected job from jenkins
//@router / [put]
func (j *JenkinsController) Put(){
	Init()
	var newjob rest.PostJob
	fmt.Println("the json string is ") //FIXME
	fmt.Println(string(j.Ctx.Input.RequestBody)) //FIXME
	json.Unmarshal(j.Ctx.Input.RequestBody, &newjob)
	fmt.Println("below is the ssh command ")
	fmt.Println(newjob.Command)



	var oldName = newjob.OldName
	oldJob,err := jenkins.GetJob(oldName)
	if err!=nil {
		fmt.Println("the job you requested does not exist")
	}
	if oldName != newjob.Name {
		oldJob.Rename(newjob.Name)
	}

	//更新完名字job,现在只需要把 nameUpdatedJob 的配置文件 根据 newjob的设置进行更新
	var nameUpdatedJob,err2 = jenkins.GetJob(newjob.Name)
	if err2!=nil {
		panic("get nameUpdated job err")
	}

	var configString string
	configString += `<?xml version="1.0" encoding="UTF-8"?>
	<maven2-moduleset plugin="maven-plugin@2.16">
		<actions></actions>
		<description>`+newjob.Description+`</description>
		<keepDependencies>false</keepDependencies>
		<properties>`

	//build discarder property of jenkins config.
	if newjob.DiscardOld {
	configString+=`<jenkins.model.BuildDiscarderProperty>
				<strategy class="hudson.tasks.LogRotator">
					<daysToKeep>`+strconv.Itoa(newjob.DaysToKeep)+`</daysToKeep>
					<numToKeep>`+strconv.Itoa(newjob.NumToKeep)+`</numToKeep>
					<artifactDaysToKeep>-1</artifactDaysToKeep>
					<artifactNumToKeep>-1</artifactNumToKeep>
				</strategy>
			</jenkins.model.BuildDiscarderProperty>`
	}
	configString+=`</properties>
		<scm class="hudson.plugins.git.GitSCM" plugin="git@3.3.0">
			<configVersion>2</configVersion>
			<userRemoteConfigs>
				<hudson.plugins.git.UserRemoteConfig>
					<url>`+newjob.GitUrl+`</url>
					<credentialsId>`+newjob.GitAuth+`</credentialsId>
				</hudson.plugins.git.UserRemoteConfig>
			</userRemoteConfigs>
			<branches>`
	// 循环增加 git branch 名字
	for j:=0 ; j<len(newjob.BranchName) ; j++{
	configString+=`<hudson.plugins.git.BranchSpec>
					<name>`+newjob.BranchName[j]+`</name>
				</hudson.plugins.git.BranchSpec>`
	}
	configString+=`</branches>
			<doGenerateSubmoduleConfigurations>false</doGenerateSubmoduleConfigurations>
			<submoduleCfg class="list"/>
			<extensions/>
		</scm>
		<canRoam>true</canRoam>
		<disabled>false</disabled>
		<blockBuildWhenDownstreamBuilding>false</blockBuildWhenDownstreamBuilding>
		<blockBuildWhenUpstreamBuilding>false</blockBuildWhenUpstreamBuilding>
		<triggers>
			<hudson.triggers.SCMTrigger>
				<spec>`+newjob.TriggerSpec+`</spec>
				<ignorePostCommitHooks>`+BoolToString(newjob.IgnorePostCommitHooks)+`</ignorePostCommitHooks>
			</hudson.triggers.SCMTrigger>
		</triggers>
		<concurrentBuild>false</concurrentBuild>`

	//如果job 有Discarder property, 则在config xml中加入相关字段
	if newjob.RootPom != "pom.xml" {
	configString+="<rootPOM>"+newjob.RootPom+"</rootPOM>"
	}
	if newjob.PomGoal != "" {
	configString+="<goals>"+newjob.PomGoal+"</goals>"
	}
	configString += `<aggregatorStyleBuild>true</aggregatorStyleBuild>
		<incrementalBuild>false</incrementalBuild>
		<ignoreUpstremChanges>false</ignoreUpstremChanges>
		<ignoreUnsuccessfulUpstreams>false</ignoreUnsuccessfulUpstreams>
		<archivingDisabled>false</archivingDisabled>
		<siteArchivingDisabled>false</siteArchivingDisabled>
		<fingerprintingDisabled>false</fingerprintingDisabled>
		<resolveDependencies>false</resolveDependencies>
		<processPlugins>false</processPlugins>
		<mavenValidationLevel>0</mavenValidationLevel>
		<runHeadless>false</runHeadless>
		<disableTriggerDownstreamProjects>false</disableTriggerDownstreamProjects>
		<blockTriggerWhenBuilding>false</blockTriggerWhenBuilding>
		<settings class="jenkins.mvn.DefaultSettingsProvider"/>
		<globalSettings class="jenkins.mvn.DefaultGlobalSettingsProvider"/>
		 <reporters>
		 <hudson.maven.reporters.MavenMailer>
	      <recipients>`+newjob.Recipients+`</recipients>
	      <dontNotifyEveryUnstableBuild>false</dontNotifyEveryUnstableBuild>
	      <sendToIndividuals>false</sendToIndividuals>
	      <perModuleEmail>true</perModuleEmail>
	    </hudson.maven.reporters.MavenMailer>
	  	</reporters>
		<publishers/>
		<buildWrappers/>
		<prebuilders/>
		<postbuilders>
			<hudson.tasks.Shell>
				<command>`+ProcessLocalCommand(newjob.Command)+`</command>
			</hudson.tasks.Shell>
			<org.jvnet.hudson.plugins.SSHBuilder plugin="ssh@2.4">
	      		<siteName>`+newjob.SiteName+`</siteName>
	      		<command>`+newjob.SshCommand+`</command>
	    	</org.jvnet.hudson.plugins.SSHBuilder>
		</postbuilders>
		<runPostStepsIfResult>
			<name></name>
			<ordinal></ordinal>
			<color></color>
			<completeBuild></completeBuild>
		</runPostStepsIfResult>
	</maven2-moduleset>`


	//var latestJob, _ = 	jenkins.GetJob(job.Name)

	result := nameUpdatedJob.UpdateConfig(configString)
	fmt.Println("belowis the result")
	fmt.Println(result)

	if result != nil{
		beego.BeeLogger.Error("modify job error ")
		j.Data["json"] = map[string]string{"error": "edit job failure"}
	}else {

		fmt.Println("below is the config string " + configString)
		j.Data["json"] = map[string]string{"name": newjob.Name}
	}
	j.ServeJSON()

}

func Init(){
	jenkins = gojenkins.CreateJenkins("http://10.20.0.99:8089/", "admin", "5f84ab66ae4444dd83dcc29abe04e8a4")
	var _, err = jenkins.Init()
	if err != nil {
		panic("Something Went Wrong while initializing jenkins")
	}
}

