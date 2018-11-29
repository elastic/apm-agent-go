#!/usr/bin/env groovy

pipeline {
  agent any
  environment {
    HOME = "${env.HUDSON_HOME}"
    BASE_DIR="src/go.elastic.co/apm"
    JOB_GIT_CREDENTIALS = "f6c7695a-671e-4f4f-a331-acdce44ff9ba"
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
    booleanParam(name: 'Run_As_Master_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on master branch.')
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
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
        GOPATH = "${env.WORKSPACE}"
      }
      steps {
        withEnvWrapper() {
          dir("${BASE_DIR}"){
            script{
              if(!env?.branch_specifier){
                echo "Checkout SCM"
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
            }
          }
          stash allowEmpty: true, name: 'source', useDefaultExcludes: false
        }
      }
    }
    /**
    Build on a linux environment.
    */
    stage('build') {
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
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
            PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
            GOPATH = "${env.WORKSPACE}"
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
                environment name: 'Run_As_Master_Branch', value: 'true'
              }
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
            PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
            GOPATH = "${env.WORKSPACE}"
          }
          when {
            beforeAgent true
            environment name: 'integration_test_ci', value: 'true'
          }
          steps {
            withEnvWrapper() {
              unstash 'source'
              dir("${BASE_DIR}"){
                sh """#!/bin/bash
                ./scripts/jenkins/docker-test.sh
                """
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
        /**
         run Go integration test with the commit version on master branch.
        */
        stage('Integration test master') {
          agent { label 'linux && immutable' }
          options { skipDefaultCheckout() }
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
                environment name: 'Run_As_Master_Branch', value: 'true'
              }
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
          options { skipDefaultCheckout() }
          when {
            beforeAgent true
            allOf {
              changeRequest()
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
    /**
      Build the documenattions.
    */
    stage('Documentation') {
      agent { label 'linux && immutable' }
      options { skipDefaultCheckout() }
      environment {
        PATH = "${env.PATH}:${env.HUDSON_HOME}/go/bin/:${env.WORKSPACE}/bin"
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
            environment name: 'Run_As_Master_Branch', value: 'true'
          }
          environment name: 'doc_ci', value: 'true'
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
