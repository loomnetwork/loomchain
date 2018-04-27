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
              ls -l
              pwd
              ./jenkins.sh
            '''
          }
        }
        stage ('Windows') {
          agent { label 'windows' }
          steps {
            sh '''
              ls -l
              pwd
              ./jenkins.sh
            '''
          }
        }
        stage ('OSX') {
          agent { label 'osx' }
          steps {
            sh '''
              ls -l
              pwd
              ./jenkins.sh
            '''
          }
        }
      }
    }
    
//     stage ('Build') {
//       def builders = [:]
//       for (x in labels) {
//         def label = x // Need to bind the label variable before the closure - can't do 'for (label in labels)'
// 
//         // Create a map to pass in to the 'parallel' step so we can fire all the builds at once
//         builders[label] = {
//           node(label) {
//             if (label!='windows') {
//               sh '''
//                   ls -l
//                   pwd
//                   ./jenkins.sh
//               '''
//             } else {
//               sh '''
//                   ./jenkins.bat
//               '''
//             }
//           }
//         }
//       }
// 
//       parallel builders
//     }
  }
}
