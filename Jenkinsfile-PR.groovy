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
  node('linux') {
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
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git',
            refspec: '+refs/pull/*/head:refs/remotes/origin/pull/*/head'
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
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "Linux");
      }
    }
  }
}

disabled['windows'] = {
  node('windows') {
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
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git',
            refspec: '+refs/pull/*/head:refs/remotes/origin/pull/*/head'
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

builders['osx'] = {
  node('osx') {
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
              ]
            ]],
          submoduleCfg: [],
          userRemoteConfigs:
          [[
            credentialsId: 'loom-sdk',
            url: 'git@github.com:loomnetwork/loomchain.git',
            refspec: '+refs/pull/*/head:refs/remotes/origin/pull/*/head'
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
      }
      else if (currentBuild.currentResult == 'SUCCESS') {
        setBuildStatus("Build ${env.BUILD_DISPLAY_NAME} succeeded in ${currentBuild.durationString.replace(' and counting', '')}", "SUCCESS", "OSX");
      }
    }
  }
}

throttle(['loom-sdk']) {
  parallel builders
}
