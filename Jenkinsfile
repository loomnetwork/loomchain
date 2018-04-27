// Inspired from https://jenkins.io/doc/pipeline/examples/

def labels = ['linux', 'windows', 'osx'] // labels for Jenkins node types we will build on

agent none

pipeline {
  stages {
    stage ('Checkout') {
      agent none
      
      parallel {
        stage ('Linux') {
          agent { label 'linux' }
          steps {
            def scmVars = checkout scm
            def commitHash = checkout(scm).GIT_COMMIT
          }
        }
        stage ('Windows') {
          agent { label 'windows' }
          steps {
            def scmVars = checkout scm
            def commitHash = checkout(scm).GIT_COMMIT
          }
        }
        stage ('OSX') {
          agent { label 'osx' }
          steps {
            def scmVars = checkout scm
            def commitHash = checkout(scm).GIT_COMMIT
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
