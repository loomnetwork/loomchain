// Inspired from https://jenkins.io/doc/pipeline/examples/

pipeline {
  agent none

  stages {
    stage ('Checkout') {
      parallel {
        stage ('Checkout - Linux') {
          agent { label 'linux' }
          steps {
            checkout scm
          }
        }
        stage ('Checkout - Windows') {
          agent { label 'windows' }
          steps {
            checkout scm
          }
        }
        stage ('Checkout - OSX') {
          agent { label 'osx' }
          steps {
            checkout scm
          }
        }
      }
    }

    stage ('Build') {
      parallel {
        stage ('Build - Linux') {
          agent { label 'linux' }
          steps {
            sh '''
              ./jenkins.sh
              cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom
            '''
          }
        }
        stage ('Build - Windows') {
          agent { label 'windows' }
          steps {
            bat '''
              jenkins.cmd
              cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/loom/windows/build-$BUILD_NUMBER/loom
            '''
          }
        }
        stage ('Build - OSX') {
          agent { label 'osx' }
          steps {
            sh '''
              ./jenkins.sh
              cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom
            '''
          }
        }
      }
    }

  }
}
