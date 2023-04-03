#!/usr/bin/env groovy

@Library('apm@current') _

pipeline {
  agent { label 'linux && immutable' }
  environment {
    REPO = 'apm-agent-go'
    BASE_DIR = "src/go.elastic.co/apm"
    NOTIFY_TO = credentials('notify-to')
    JOB_GCS_BUCKET = credentials('gcs-bucket')
    GO111MODULE = 'on'
    GOPATH = "${env.WORKSPACE}"
    GOPROXY = 'https://proxy.golang.org'
    HOME = "${env.WORKSPACE}"
    SLACK_CHANNEL = '#apm-agent-go'
  }
  options {
    timeout(time: 2, unit: 'HOURS')
    buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20', daysToKeepStr: '30'))
    timestamps()
    ansiColor('xterm')
    disableResume()
    durabilityHint('PERFORMANCE_OPTIMIZED')
    rateLimitBuilds(throttle: [count: 60, durationName: 'hour', userBoost: true])
    quietPeriod(10)
  }
  triggers {
    issueCommentTrigger("^run benchmark tests")
  }
  parameters {
    string(name: 'GO_VERSION', defaultValue: "1.15.10", description: "Go version to use.")
    booleanParam(name: 'Run_As_Main_Branch', defaultValue: false, description: 'Allow to run any steps on a PR, some steps normally only run on main branch.')
    booleanParam(name: 'bench_ci', defaultValue: true, description: 'Enable benchmarks')
  }
  stages {
    stage('Initializing'){
      options { skipDefaultCheckout() }
      environment {
        GO_VERSION = "${params.GO_VERSION}"
        PATH = "${env.PATH}:${env.WORKSPACE}/bin"
      }
      when {
        beforeAgent true
        allOf {
          anyOf {
            branch 'main'
            tag pattern: 'v\\d+\\.\\d+\\.\\d+.*', comparator: 'REGEXP'
            expression { return params.Run_As_Main_Branch }
            expression { return env.GITHUB_COMMENT?.contains('benchmark tests') }
          }
          expression { return params.bench_ci }
        }
      }
      stages {
        /**
         Checkout the code and stash it, to use it on other stages.
        */
        stage('Checkout') {
          options { skipDefaultCheckout() }
          steps {
            pipelineManager([ cancelPreviousRunningBuilds: [ when: 'PR' ] ])
            deleteDir()
            gitCheckout(basedir: "${BASE_DIR}", githubNotifyFirstTimeContributor: true, reference: '/var/lib/jenkins/.git-references/apm-agent-go.git')
            stash allowEmpty: true, name: 'source', useDefaultExcludes: false
          }
        }
        stage('Benchmark') {
          agent { label 'microbenchmarks-pool' }
          options { skipDefaultCheckout() }
          steps {
            withGithubNotify(context: 'Benchmark', tab: 'tests') {
              deleteDir()
              unstash 'source'
              dir("${BASE_DIR}"){
                sh script: './scripts/jenkins/bench.sh', label: 'Benchmarking'
                dir('build') {
                  sendBenchmarks(file: 'bench.out', index: 'benchmark-go')
                  generateGoBenchmarkDiff(file: 'bench.out', filter: 'exclude')
                }
              }
            }
          }
        }
      }
    }
  }
}
