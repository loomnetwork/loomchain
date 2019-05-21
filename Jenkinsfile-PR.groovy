void setBuildStatus(String message, String state, String context) {
  step([
      $class: "GitHubCommitStatusSetter",
      reposSource: [$class: "ManuallyEnteredRepositorySource", url: "git@github.com:loomnetwork/loomchain.git"],
      contextSource: [$class: "ManuallyEnteredCommitContextSource", context: context],
      errorHandlers: [[$class: "ChangingBuildStatusErrorHandler", result: "UNSTABLE"]],
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
            branches: [[name: 'origin/pull/*/head']],
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

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Linux");

        stage ('Build - Linux') {
          sh '''
            ./jenkins.sh
          '''
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Linux");
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-linux-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux");
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
            branches: [[name: 'origin/pull/*/head']],
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

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "Windows");

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
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "Windows");
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Windows");
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
            branches: [[name: 'origin/pull/*/head']],
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

        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} is in progress", "PENDING", "OSX");

        stage ('Build - OSX') {
          sh '''
            ./jenkins.sh
          '''
        }
      } catch (e) {
        thisBuild = 'FAILURE'
        throw e
      } finally {
        if (currentBuild.currentResult == 'FAILURE' || thisBuild == 'FAILURE') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} failed", "FAILURE", "OSX");
          sh '''
            cd /tmp/gopath-jenkins-${JOB_BASE_NAME}-${BUILD_NUMBER}/src/github.com/loomnetwork/loomchain/e2e
            find test-data -name "*.log" | tar -czf ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz -T -
            mkdir -p /tmp/test-data
            mv ${JOB_BASE_NAME}-${BUILD_NUMBER}-osx-test-data.tar.gz /tmp/test-data
          '''
        }
        else if (currentBuild.currentResult == 'SUCCESS') {
          setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX");
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
