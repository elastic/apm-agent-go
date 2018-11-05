#!/usr/bin/env groovy

library identifier: 'apm@master', 
retriever: modernSCM(
  [$class: 'GitSCMSource', 
  credentialsId: 'f6c7695a-671e-4f4f-a331-acdce44ff9ba', 
  remote: 'git@github.com:elastic/apm-pipeline-library.git'])

pipeline {
  agent any
  environment {
    HOME = "${env.HUDSON_HOME}"
    BASE_DIR="src/go.elastic.co/apm"
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
  }
  triggers {
    cron('0 0 * * 1-5')
    githubPush()
  }
  options {
    timeout(time: 1, unit: 'HOURS') 
    buildDiscarder(logRotator(numToKeepStr: '3', artifactNumToKeepStr: '2', daysToKeepStr: '30'))
    timestamps()
    preserveStashes()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
  }
  parameters {
    string(name: 'branch_specifier', defaultValue: "", description: "the Git branch specifier to build (<branchName>, <tagName>, <commitId>, etc.)")
    string(name: 'GO_VERSION', defaultValue: "1.10.3", description: "Go version to use.")
    booleanParam(name: 'linux_ci', defaultValue: true, description: 'Enable Linux build')
    booleanParam(name: 'test_ci', defaultValue: true, description: 'Enable test')
    booleanParam(name: 'integration_test_ci', defaultValue: true, description: 'Enable run integration test')
    booleanParam(name: 'integration_test_pr_ci', defaultValue: false, description: 'Enable run integration test')
    booleanParam(name: 'integration_test_master_ci', defaultValue: false, description: 'Enable run integration test')
    booleanParam(name: 'bench_ci', defaultValue: true, description: 'Enable benchmarks')
    booleanParam(name: 'doc_ci', defaultValue: true, description: 'Enable build documentation')
  }
  
  stages {
    /**
     Checkout the code and stash it, to use it on other stages.
    */
    stage('Checkout') {
      agent { label 'master || linux' }
      environment {
        PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
        GOPATH = "${env.WORKSPACE}"
      }
      
      steps {
        withEnvWrapper() {
          dir("${BASE_DIR}"){
            script{
              if(!branch_specifier){
                echo "Checkout SCM ${GIT_BRANCH}"
                checkout scm
              } else {
                echo "Checkout ${branch_specifier}"
                checkout([$class: 'GitSCM', branches: [[name: "${branch_specifier}"]], 
                  doGenerateSubmoduleConfigurations: false, 
                  extensions: [],
                  submoduleCfg: [],
                  userRemoteConfigs: [[credentialsId: "${JOB_GIT_CREDENTIALS}", 
                  url: "${GIT_URL}"]]])
              }
              env.JOB_GIT_COMMIT = getGitCommitSha()
              env.JOB_GIT_URL = "${GIT_URL}"
              
              github_enterprise_constructor()
              
              on_change{
                echo "build cause a change (commit or PR)"
              }
              
              on_commit {
                echo "build cause a commit"
              }
              
              on_merge {
                echo "build cause a merge"
              }
              
              on_pull_request {
                echo "build cause PR"
              }
            }
          }
          stash allowEmpty: true, name: 'source'
        }
      }
    }
    
    /**
    Build on a linux environment.
    */
    stage('build') { 
      agent { label 'linux && immutable' }
      environment {
        PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
        GOPATH = "${env.WORKSPACE}"
      }
      
      when { 
        beforeAgent true
        environment name: 'linux_ci', value: 'true' 
      }
      steps {
        withEnvWrapper() {
          unstash 'source'
          dir("${BASE_DIR}"){    
            sh """#!/bin/bash
            ./scripts/jenkins/build.sh
            """
          }
        }
      }
    }
    stage('Parallel stages') {
      failFast true
      parallel {
        stage('test') {
          agent { label 'linux && immutable' }
          environment {
            PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
            GOPATH = "${env.WORKSPACE}"
          }
          
          when { 
            beforeAgent true
            environment name: 'test_ci', value: 'true' 
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){    
                sh """#!/bin/bash
                ./scripts/jenkins/test.sh
                """
                codecov('apm-agent-go')
              }
            }
          }
          post { 
            always {
              coverageReport("${BASE_DIR}/build/coverage")
              junit(allowEmptyResults: true, 
                keepLongStdio: true, 
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        stage('Benchmarks') {
          agent { label 'linux && immutable' }
          environment {
            PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
            GOPATH = "${env.WORKSPACE}"
          }
          
          when { 
            beforeAgent true
            allOf { 
              //branch 'master';
              environment name: 'bench_ci', value: 'true' 
            }
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){    
                sh """#!/bin/bash
                ./scripts/jenkins/bench.sh
                """
                sendBenchmarks(file: 'build/bench.out')
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
        stage('Docker tests') {
          agent { label 'linux && docker && immutable' }
          environment {
            PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
            GOPATH = "${env.WORKSPACE}"
          }
          
          when { 
            beforeAgent true
            allOf { 
              //branch 'master';
              environment name: 'integration_test_ci', value: 'true' 
            }
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){    
                sh """#!/bin/bash
                ./scripts/jenkins/docker-test.sh
                """
                codecov('apm-agent-go')
              }
            }
          }
          post { 
            always { 
              coverageReport("${BASE_DIR}/build/coverage")
              junit(allowEmptyResults: true, 
                keepLongStdio: true, 
                testResults: "${BASE_DIR}/build/junit-*.xml")
            }
          }
        }
        
        /**
         run Go integration test with the commit version on master branch.
        */
        stage('Integration test master') { 
          agent { label 'linux && immutable' }
          when { 
            beforeAgent true
            allOf { 
              //branch 'master';
              environment name: 'integration_test_master_ci', value: 'true' 
            }
          }
          steps {
            build(
              job: 'apm-server-ci/apm-integration-test-axis-pipeline', 
              parameters: [
                string(name: 'BUILD_DESCRIPTION', value: "${BUILD_TAG}-INTEST"),
                booleanParam(name: "go_Test", value: true),
                booleanParam(name: "java_Test", value: false),
                booleanParam(name: "ruby_Test", value: false),
                booleanParam(name: "python_Test", value: false),
                booleanParam(name: "nodejs_Test", value: false)],
              wait: true,
              propagate: true)
          }
        }
        
        /**
         run Go integration test with the commit version on a PR.
        */
        stage('Integration test PR') { 
          agent { label 'linux && immutable' }
          when { 
            beforeAgent true
            allOf { 
              not { branch 'master' };
              environment name: 'integration_test_pr_ci', value: 'true' 
            }
          }
          steps {
            build(
              job: 'apm-server-ci/apm-integration-test-pipeline', 
              parameters: [
                string(name: 'BUILD_DESCRIPTION', value: "${BUILD_TAG}-INTEST"),
                string(name: 'APM_AGENT_GO_PKG', value: "${BUILD_TAG}"),
                booleanParam(name: "go_Test", value: true),
                booleanParam(name: "java_Test", value: false),
                booleanParam(name: "ruby_Test", value: false),
                booleanParam(name: "python_Test", value: false),
                booleanParam(name: "nodejs_Test", value: false),
                booleanParam(name: "kibana_Test", value: false),
                booleanParam(name: "server_Test", value: false)],
              wait: true,
              propagate: true)
          }
        }
      }
    }
        
    stage('Documentation') { 
      agent { label 'linux && immutable' }
      environment {
        PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
        GOPATH = "${env.WORKSPACE}"
        ELASTIC_DOCS = "${env.WORKSPACE}/elastic/docs"
      }
      
      when { 
        beforeAgent true
        allOf { 
          //branch 'master';
          environment name: 'doc_ci', value: 'true' 
        }
      }
      steps {
        withEnvWrapper() {
          unstash 'source'
          dir("${ELASTIC_DOCS}"){
            git "https://github.com/elastic/docs.git"
          }
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
    always { 
      echo 'Post Actions'
    }
    success { 
      echo 'Success Post Actions'
    }
    aborted { 
      echo 'Aborted Post Actions'
    }
    failure { 
      echo 'Failure Post Actions'
      //step([$class: 'Mailer', notifyEveryUnstableBuild: true, recipients: "${NOTIFY_TO}", sendToIndividuals: false])
    }
    unstable { 
      echo 'Unstable Post Actions'
    }
  }
}
