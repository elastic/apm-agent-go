#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent any
  environment {
    REPO = 'apm-agent-go'
    BASE_DIR = "src/go.elastic.co/apm"
    NOTIFY_TO = credentials('notify-to')
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    CODECOV_SECRET = 'secret/apm-team/ci/apm-agent-go-codecov'
    GO111MODULE = 'on'
    GOPROXY = 'https://proxy.golang.org'
    GITHUB_CHECK_ITS_NAME = 'Integration Tests'
    ITS_PIPELINE = 'apm-integration-tests-selector-mbp/master'
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    issueCommentTrigger('(?i).*(?:jenkins\\W+)?run\\W+(?:the\\W+)?tests(?:\\W+please)?.*')
  }
  parameters {
    string(name: 'GO_VERSION', defaultValue: "1.12.7", description: "Go version to use.")
    booleanParam(name: 'Run_As_Master_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on master branch.')
    booleanParam(name: 'test_ci', defaultValue: true, description: 'Enable test')
    booleanParam(name: 'docker_test_ci', defaultValue: true, description: 'Enable run docker tests')
    booleanParam(name: 'bench_ci', defaultValue: true, description: 'Enable benchmarks')
  }
  stages {
    stage('Initializing'){
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
        GO_VERSION = "${params.GO_VERSION}"
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
      }
      stages {
        /**
         Checkout the code and stash it, to use it on other stages.
        */
        stage('Checkout') {
          steps {
            gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true)
            stash allowEmpty: true, name: 'source', useDefaultExcludes: false
          }
        }
        /**
        Execute unit tests.
        */
        stage('Tests') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression { return params.test_ci }
          }
          steps {
            withGithubNotify(context: 'Tests', tab: 'tests') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                script {
                  def go = readYaml(file: '.jenkins.yml')
                  def parallelTasks = [:]
                  go['GO_VERSION'].each{ version ->
                    parallelTasks["Go-${version}"] = generateStep(version)
                  }
                  // For the cutting edge
                  def edge = readYaml(file: '.jenkins-edge.yml')
                  edge['GO_VERSION'].each{ version ->
                    parallelTasks["Go-${version}"] = generateStepAndCatchError(version)
                  }
                  parallel(parallelTasks)
                }
              }
            }
          }
        }
        stage('Coverage') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            expression { return params.docker_test_ci }
          }
          steps {
            withGithubNotify(context: 'Coverage') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
                sh script: './scripts/jenkins/docker-test.sh', label: 'Docker tests'
              }
            }
          }
          post {
            always {
              coverageReport("${BASE_DIR}/build/coverage")
              codecov(repo: env.REPO, basedir: "${BASE_DIR}",
                flags: "-f build/coverage/coverage.cov -X search",
                secret: "${CODECOV_SECRET}")
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        stage('Benchmark') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            allOf {
              anyOf {
                branch 'master'
                branch "\\d+\\.\\d+"
                branch "v\\d?"
                tag "v\\d+\\.\\d+\\.\\d+*"
                expression { return params.Run_As_Master_Branch }
              }
              expression { return params.bench_ci }
            }
          }
          steps {
            withGithubNotify(context: 'Benchmark', tab: 'tests') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
                sh script: './scripts/jenkins/bench.sh', label: 'Benchmarking'
                sendBenchmarks(file: 'build/bench.out', index: 'benchmark-go')
              }
            }
          }
        }
      }
    }
    stage('Windows') {
      agent { label 'windows' }
      options { skipDefaultCheckout() }
      environment {
        GOROOT = "c:\\Go"
        GOPATH = "${env.WORKSPACE}"
        PATH = "${env.PATH};${env.GOROOT}\\bin;${env.GOPATH}\\bin"
        GO_VERSION = "${params.GO_VERSION}"
      }
      steps {
        withGithubNotify(context: 'Build-Test - Windows') {
          cleanDir("${WORKSPACE}/${BASE_DIR}")
          unstash 'source'
          dir("${BASE_DIR}"){
            bat script: 'scripts/jenkins/windows/install-tools.bat', label: 'Install tools'
            bat script: 'scripts/jenkins/windows/build-test.bat', label: 'Build and test'
          }
        }
      }
      post {
        always {
          junit(allowEmptyResults: true, keepLongStdio: true, testResults: "${BASE_DIR}/build/junit-*.xml")
          dir("${BASE_DIR}"){
            bat script: 'scripts/jenkins/windows/uninstall-tools.bat', label: 'Uninstall tools'
          }
          cleanWs(disableDeferredWipeout: true, notFailBuild: true)
        }
      }
    }
    stage('OSX') {
      agent none
      /** TODO: As soon as MacOSX are available we will provide the stage implementation */
      when {
        beforeAgent true
        expression { return false }
      }
      steps {
        echo 'TBD'
      }
    }
    stage('Integration Tests') {
      agent none
      when {
        beforeAgent true
        allOf {
          anyOf {
            environment name: 'GIT_BUILD_CAUSE', value: 'pr'
            expression { return !params.Run_As_Master_Branch }
          }
        }
      }
      steps {
        log(level: 'INFO', text: 'Launching Async ITs')
        build(job: env.ITS_PIPELINE, propagate: false, wait: false,
              parameters: [string(name: 'AGENT_INTEGRATION_TEST', value: 'Go'),
                           string(name: 'BUILD_OPTS', value: "--go-agent-version ${env.GIT_BASE_COMMIT}"),
                           string(name: 'GITHUB_CHECK_NAME', value: env.GITHUB_CHECK_ITS_NAME),
                           string(name: 'GITHUB_CHECK_REPO', value: env.REPO),
                           string(name: 'GITHUB_CHECK_SHA1', value: env.GIT_BASE_COMMIT)])
        githubNotify(context: "${env.GITHUB_CHECK_ITS_NAME}", description: "${env.GITHUB_CHECK_ITS_NAME} ...", status: 'PENDING', targetUrl: "${env.JENKINS_URL}search/?q=${env.ITS_PIPELINE.replaceAll('/','+')}")
      }
    }
  }
  post {
    cleanup {
      notifyBuildResult()
    }
  }
}

def generateStep(version){
  return {
    node('docker && linux && immutable'){
      try {
        deleteDir()
        unstash 'source'
        echo "${version}"
        dir("${BASE_DIR}"){
          withEnv([
            "GO_VERSION=${version}",
            "HOME=${WORKSPACE}"]) {
            sh script: './scripts/jenkins/before_install.sh', label: 'Install dependencies'
            sh script: './scripts/jenkins/build-test.sh', label: 'Build and test'
          }
        }
      } catch(e){
        error(e.toString())
      } finally {
        junit(allowEmptyResults: true,
          keepLongStdio: true,
          testResults: "${BASE_DIR}/build/junit-*.xml")
      }
    }
  }
}

def generateStepAndCatchError(version){
  return {
    catchError(buildResult: 'SUCCESS', message: 'Cutting Edge Tests', stageResult: 'UNSTABLE') {
      generateStep(version)
    }
  }
}

def cleanDir(path){
  powershell label: "Clean ${path}", script: "Remove-Item -Recurse -Force ${path}"
}
