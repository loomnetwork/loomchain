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
              cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom
            '''
          }
        }
        stage ('Windows') {
          agent { label 'windows' }
          steps {
            bat '''
              jenkins.cmd
              cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
              gsutil cp loom gs://private.delegatecall.com/loom/windows/build-$BUILD_NUMBER/loom
            '''
          }
        }
        stage ('OSX') {
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
