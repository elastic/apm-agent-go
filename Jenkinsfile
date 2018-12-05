#!/usr/bin/env groovy

pipeline {
  agent { label 'master' }
  environment {
    BASE_DIR="src/go.elastic.co/apm"
  }
  options {
    timeout(time: 1, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  parameters {
    string(name: 'GO_VERSION', defaultValue: "1.10.3", description: "Go version to use.")
    booleanParam(name: 'Run_As_Master_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on master branch.')
    booleanParam(name: 'test_ci', defaultValue: true, description: 'Enable test')
    booleanParam(name: 'docker_test_ci', defaultValue: true, description: 'Enable run docker tests')
    booleanParam(name: 'bench_ci', defaultValue: true, description: 'Enable benchmarks')
    booleanParam(name: 'doc_ci', defaultValue: true, description: 'Enable build documentation')
  }
  stages {
    stage('Dummy'){
      agent { label 'master' }
      options { skipDefaultCheckout() }
      steps {
        sh 'export'
        checkout([$class: 'GitSCM', 
          branches: [[name: "${env?.CHANGE_ID ? env?.GIT_COMMIT : env?.BRANCH_NAME}"]],
          doGenerateSubmoduleConfigurations: false, 
          extensions: [
            [$class: 'ChangelogToBranch', 
              options: [compareRemote: "${env?.GIT_URL}", 
              compareTarget: "${env?.CHANGE_ID ? env?.CHANGE_TARGET : 'master'}"]],
            [$class: 'DisableRemotePoll'],
            [$class: 'CloneOption', 
              noTags: false, 
              reference: '/var/lib/jenkins/.git-references/apm-agent-go.git', 
              shallow: false]], 
          submoduleCfg: [], 
          userRemoteConfigs: [
            [credentialsId: '2a9602aa-ab9f-4e52-baf3-b71ca88469c7-UserAndToken', 
            url: "${env?.GIT_URL}"]]])
        error "Please do not continue"
      }
    }
    stage('Initializing'){
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
        GO_VERSION = "${params.GO_VERSION}"
      }
      stages {
        /**
         Checkout the code and stash it, to use it on other stages.
        */
        stage('Checkout') {
          steps {
            gitCheckout(basedir: "${BASE_DIR}")
            stash allowEmpty: true, name: 'source', useDefaultExcludes: false
          }
        }
        /**
        Build on a linux environment.
        */
        stage('build') {
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh './scripts/jenkins/build.sh'
              }
            }
          }
        }
      }
    }
    stage('Test') {
      failFast true
      parallel {
        /**
          Run unit tests and store the results in Jenkins.
        */
        stage('Unit Test') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          environment {
            PATH = "${env.PATH}:${env.WORKSPACE}/bin"
            HOME = "${env.WORKSPACE}"
            GOPATH = "${env.WORKSPACE}"
            GO_VERSION = "${params.GO_VERSION}"
          }
          when {
            beforeAgent true
            expression { return params.test_ci }
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh './scripts/jenkins/test.sh'
              }
            }
          }
          post {
            always {
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        /**
          Run Benchmarks and send the results to ES.
        */
        stage('Benchmarks') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
          environment {
            PATH = "${env.PATH}:${env.WORKSPACE}/bin"
            HOME = "${env.WORKSPACE}"
            GOPATH = "${env.WORKSPACE}"
            GO_VERSION = "${params.GO_VERSION}"
          }
          when {
            beforeAgent true
            allOf {
              anyOf {
                not {
                  changeRequest()
                }
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
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh './scripts/jenkins/bench.sh'
                sendBenchmarks(file: 'build/bench.out', index: "benchmark-go")
              }
            }
          }
          post {
            always {
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        /**
          Run tests in a docker container and store the results in jenkins and codecov.
        */
        stage('Docker tests') {
          agent { label 'linux && docker && immutable' }
          options { skipDefaultCheckout() }
          environment {
            PATH = "${env.PATH}:${env.WORKSPACE}/bin"
            HOME = "${env.WORKSPACE}"
            GOPATH = "${env.WORKSPACE}"
            GO_VERSION = "${params.GO_VERSION}"
          }
          when {
            beforeAgent true
            expression { return params.docker_test_ci }
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh './scripts/jenkins/docker-test.sh'
              }
            }
          }
          post {
            always {
              coverageReport("${BASE_DIR}/build/coverage")
              junit(allowEmptyResults: true,
                keepLongStdio: true,
                testResults: "${BASE_DIR}/build/junit-*.xml")
              codecov(repo: 'apm-agent-go', basedir: "${BASE_DIR}")
            }
          }
        }
      }
    }
    /**
      Build the documentation.
    */
    stage('Documentation') {
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
        HOME = "${env.WORKSPACE}"
        GOPATH = "${env.WORKSPACE}"
        ELASTIC_DOCS = "${env.WORKSPACE}/elastic/docs"
      }
      when {
        beforeAgent true
        allOf {
          anyOf {
            not {
              changeRequest()
            }
            branch 'master'
            branch "\\d+\\.\\d+"
            branch "v\\d?"
            tag "v\\d+\\.\\d+\\.\\d+*"
            expression { return params.Run_As_Master_Branch }
          }
          expression { return params.doc_ci }
        }
      }
      steps {
        withEnvWrapper() {
          unstash 'source'
          checkoutElasticDocsTools(basedir: "${ELASTIC_DOCS}")
          dir("${BASE_DIR}"){
            sh """#!/bin/bash
            make docs
            """
          }
        }
      }
      post{
        success {
          tar(file: "doc-files.tgz", archive: true, dir: "html", pathPrefix: "${BASE_DIR}/docs")
        }
      }
    }
  }
  post {
    success {
      echoColor(text: '[SUCCESS]', colorfg: 'green', colorbg: 'default')
    }
    aborted {
      echoColor(text: '[ABORTED]', colorfg: 'magenta', colorbg: 'default')
    }
    failure {
      echoColor(text: '[FAILURE]', colorfg: 'red', colorbg: 'default')
      //step([$class: 'Mailer', notifyEveryUnstableBuild: true, recipients: "${NOTIFY_TO}", sendToIndividuals: false])
    }
    unstable {
      echoColor(text: '[UNSTABLE]', colorfg: 'yellow', colorbg: 'default')
    }
  }
}
