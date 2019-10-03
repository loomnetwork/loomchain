void setBuildStatus(String message, String state, String context, String sha1) {
  step([
      $class: "GitHubCommitStatusSetter",
      reposSource: [$class: "ManuallyEnteredRepositorySource", url: "git@github.com:loomnetwork/loomchain.git"],
      contextSource: [$class: "ManuallyEnteredCommitContextSource", context: context],
      errorHandlers: [[$class: "ChangingBuildStatusErrorHandler", result: "UNSTABLE"]],
      commitShaSource: [$class: 'ManuallyEnteredShaSource', sha: sha1],
      statusResultSource: [ $class: "ConditionalStatusResultSource", results: [[$class: "AnyBuildResult", message: message, state: state]] ]
  ]);
}

def builders = [:]
def disabled = [:]

builders['linux'] = {
  node('linux-any') {
    timestamps {
      def thisBuild = null

      try {
        stage ('Checkout - Linux') {
          checkout changelog: true, poll: true, scm:
          [
            $class: 'GitSCM',
            branches: [[name: '${ghprbActualCommit}']],
            doGenerateSubmoduleConfigurations: false,
            extensions: [
              [$class: 'PreBuildMerge',
              options: [
                fastForwardMode: 'FF',
                mergeRemote: 'origin',
                mergeTarget: 'master'
                ]],
              [$class: 'CleanBeforeCheckout'],
              [$class: 'PruneStaleBranch']
              ],
            submoduleCfg: [],
            userRemoteConfigs:
            [[
              credentialsId: 'loom-sdk',
              url: 'git@github.com:loomnetwork/loomchain.git',
              refspec: '+refs/heads/master:refs/remotes/origin/master +refs/pull/*/head:refs/remotes/origin/pull/*/head'
            ]]
          ]
        }

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Linux", "${ghprbActualCommit}");

        stage ('Build - Linux') {
          nodejs('v10.16.3 (LTS)') {
            sh '''
              ./jenkins.sh
            '''
            }
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Linux", "${ghprbActualCommit}");
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux", "${ghprbActualCommit}");
        }
      }
    }
  }
}

disabled['windows'] = {
  node('windows-any') {
      timestamps {
      def thisBuild = null

      try {
        stage ('Checkout - Windows') {
          checkout changelog: true, poll: true, scm:
          [
            $class: 'GitSCM',
            branches: [[name: '${ghprbActualCommit}']],
            doGenerateSubmoduleConfigurations: false,
            extensions: [
              [$class: 'PreBuildMerge',
              options: [
                fastForwardMode: 'FF',
                mergeRemote: 'origin',
                mergeTarget: 'master'
                ]],
              [$class: 'CleanBeforeCheckout'],
              [$class: 'PruneStaleBranch']
              ],
            submoduleCfg: [],
            userRemoteConfigs:
            [[
              credentialsId: 'loom-sdk',
              url: 'git@github.com:loomnetwork/loomchain.git',
              refspec: '+refs/heads/master:refs/remotes/origin/master +refs/pull/*/head:refs/remotes/origin/pull/*/head'
            ]]
          ]
        }

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Windows", "${ghprbActualCommit}");

        stage ('Build - Windows') {
          bat '''
            jenkins.cmd
          '''
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Windows", "${ghprbActualCommit}");
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Windows", "${ghprbActualCommit}");
        }
      }
    }
  }
}

builders['osx'] = {
  node('osx-any') {
    timestamps {
      def thisBuild = null

      try {
        stage ('Checkout - OSX') {
          checkout changelog: true, poll: true, scm:
          [
            $class: 'GitSCM',
            branches: [[name: '${ghprbActualCommit}']],
            doGenerateSubmoduleConfigurations: false,
            extensions: [
              [$class: 'PreBuildMerge',
              options: [
                fastForwardMode: 'FF',
                mergeRemote: 'origin',
                mergeTarget: 'master'
                ]],
              [$class: 'CleanBeforeCheckout'],
              [$class: 'PruneStaleBranch']
              ],
            submoduleCfg: [],
            userRemoteConfigs:
            [[
              credentialsId: 'loom-sdk',
              url: 'git@github.com:loomnetwork/loomchain.git',
              refspec: '+refs/heads/master:refs/remotes/origin/master +refs/pull/*/head:refs/remotes/origin/pull/*/head'
            ]]
          ]
        }

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "OSX", "${ghprbActualCommit}");

        stage ('Build - OSX') {
          nodejs('v10.16.3 (LTS)') {
            sh '''
              ./jenkins.sh
            '''
          }
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "OSX", "${ghprbActualCommit}");
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX", "${ghprbActualCommit}");
        }
      }
    }
  }
}

throttle(['loom-sdk']) {
  timeout(time: 60, unit: 'MINUTES'){
    parallel builders
  }
}
