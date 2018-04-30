// Inspired from https://jenkins.io/doc/pipeline/examples/

def labels = ['linux', 'windows', 'osx'] // labels for Jenkins node types we will build on

pipeline {
  agent none

  stages {
    stage ('Checkout') {
      parallel {
        stage ('Linux') {
          agent { label 'linux' }
          steps {
            checkout scm
          }
        }
        stage ('Windows') {
          agent { label 'windows' }
          steps {
            checkout scm
          }
        }
        stage ('OSX') {
          agent { label 'osx' }
          steps {
            checkout scm
          }
        }
      }
    }

    stage ('Build') {
      parallel {
        stage ('Linux') {
          agent { label 'linux' }
          steps {
            sh '''
              ./jenkins.sh
            '''
          }
        }
        stage ('Windows') {
          agent { label 'windows' }
          steps {
            bat '''
              jenkins.cmd
            '''
          }
        }
        stage ('OSX') {
          agent { label 'osx' }
          steps {
            sh '''
              ./jenkins.sh
              cd /tmp/gopath-jenkins-loom-sdk-pipeline-test-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/osx/build-$BUILD_NUMBER/loom
              gsutil cp ladmin gs://private.delegatecall.com/osx/build-$BUILD_NUMBER/ladmin
            '''
          }
        }
      }
    }

  }
}
