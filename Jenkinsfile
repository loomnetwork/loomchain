// Inspired from https://jenkins.io/doc/pipeline/examples/

pipeline {
  agent none

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

    parallel {
      stage ('Build - Linux') {
        agent { label 'linux' }
        steps {
          sh '''
            ./jenkins.sh
          '''
        }
      }
      stage ('Build - Windows') {
        agent { label 'windows' }
        steps {
          bat '''
            jenkins.cmd
          '''
        }
      }
      stage ('Build - OSX') {
        agent { label 'osx' }
        steps {
          sh '''
            ./jenkins.sh
          '''
        }
      }
    }

    parallel {
      stage ('Push - Linux') {
        agent { label 'linux' }
        steps {
          sh '''
            cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
            gsutil cp loom gs://private.delegatecall.com/loom/linux/build-$BUILD_NUMBER/loom
          '''
        }
      }
      stage ('Push - Windows') {
        agent { label 'windows' }
        steps {
          bat '''
            cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
            gsutil cp loom gs://private.delegatecall.com/loom/windows/build-$BUILD_NUMBER/loom
          '''
        }
      }
      stage ('Push - OSX') {
        agent { label 'osx' }
        steps {
          sh '''
            cd /tmp/gopath-${BUILD_TAG}/src/github.com/loomnetwork/loomchain/
            gsutil cp loom gs://private.delegatecall.com/loom/osx/build-$BUILD_NUMBER/loom
          '''
        }
      }
    }

  }
}
