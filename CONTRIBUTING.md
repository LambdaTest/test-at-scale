# Contributing to Test-at-scale

Thank you for your interest in Test-at-scale and for taking the time to contribute to this project. If you feel insecure about how to start contributing, feel free to ask us on our [Discord Server](https://discord.gg/Wyf8srhf6K) in the #contribute channel.

## **Code of conduct**

Read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.


## **How can I contribute?**

There are many ways in which you can contribute to Test-at-scale.

#### 👥 Join the community
&emsp;&emsp;Join our [Discord server](https://discord.gg/Wyf8srhf6K), help others use Test-at-scale for their test automation requirements.

#### 🗣️ Give a talk about Test-at-scale
&emsp;&emsp;You can talk about Test-at-scale in online/offline meetups. Drop a line to [hello.tas@lambdatest.com](mailto:hello.tas@lambdatest.com) ahead of time and we'll send you some swag. 👕

#### 🧩 Build an Add-on 
&emsp;&emsp;Enhance Test-at-scale’s capabilities by building add-ons to solve unique problems. 

#### 🐞 Report a bug
&emsp;&emsp;Report all issues through GitHub Issues and provide as much information as you can.

#### 🛠 Create a feature request
&emsp;&emsp;We welcome all feature requests, whether for new features or enhancements to existing features. File your feature request through GitHub Issues.

#### 📝 Improve the documentation
&emsp;&emsp;Suggest improvements to our documentation using the [Documentation Improvement](https://github.com/LambdaTest/test-at-scale/issues/new) template. Test-at-scale docs are published on [here](https://www.lambdatest.com/support/docs/getting-started-with-tas/)


#### 📚 Contribute to Tutorials 
&emsp;&emsp;You can help by suggesting improvements to our tutorials using the [Tutorials Improvement](https://github.com/LambdaTest/test-at-scale/issues/new) template or create a new tutorial. 


#### ⚙️ Write code to fix a Bug / new Feature Request
&emsp;&emsp;We welcome contributions that help make Test-at-scale bug-free & improve the test automation experience for our users. You can also find issues tagged [Good First Issues](https://github.com/LambdaTest/test-at-scale/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22"). Check out the below sections to begin.

&emsp;

## **Writing Code**
All submissions, including submissions by project members, require review. Before raising a pull request, ensure you have raised a corresponding issue and discussed a possible solution with a maintainer. This gives your pull request the highest chance of getting merged quickly. Join our [Discord Server](https://discord.gg/Wyf8srhf6K) if you need any help.

 
### First-time contributors
We appreciate first-time contributors and we are happy to assist you in getting started. In case of questions, just [reach out to us!](https://discord.gg/Wyf8srhf6K)
You find all issues suitable for first-time contributors [here.](https://github.com/LambdaTest/test-at-scale/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)


### Repo overview

[LambdaTest/test-at-scale](https://github.com/LambdaTest/test-at-scale/) consists of 2 components:

- **Synapse:** is the agent responsible for fetching jobs from Test at Scale servers to execute them on the self hosted environment (your laptop or your server farm). Synapse coordinates with nucleus (test runner) and TAS cloud to execute tests and push out test details such as test name, test suite, execution logs, execution metrics.
- **Test Runners:** component is the driving agent of the container executed to run the actions received by synapse. All actions will be executed on Linux containers and itself manages the lifecycle of the container. It provides functionalities such as logging, metric collections, etc. It primarily conducts two primary stages viz. test discovery and test execution. Both of these stage are accomplished by using plugins for language and framework to make sure nucleus is not tightly coupled with specific languages.
 
<details>
<summary>Read More</summary>
We've engineered the platform such that you can setup the test-runners anywhere, from your local workstation to any cloud (AWS, Azure, GCP etc), as per your convenience. 
<p align="center">
<img loading="lazy" src={require('https://www.lambdatest.com/support/assets/images/synapse-tas-interaction-a70a50f02b2e6e99491777ce636538f4.png').default} alt="Synapse Architecture" width="1340" height="617" className="doc_img"/>
</p>

When you configure TAS to run in a self-hosted environment, all the test execution jobs are executed inside your  environment. Your code stays within your setup environment. To provide you with test-insights on the TAS portal we store information only related to tests like name of testFile, testCase, testSuite and execution logs. At no point, we collect business logic of your code.


Here is a sample flow to understand how it works:
- After Configure TAS self-hosted mode and integrating your repositories into TAS platform.
- Whenever you make a commit, raise a PR or merge a PR, the TAS platform receives a webhook event from your git provider.
- This webhook event is simply sent to your self-hosted environment to initate jobs for test execution.
- The Test-at-scale binary running on your self hosted enviroment spawns containers to execute those jobs.
- Basic test metadata is sent to the TAS server to provide you with test insights and other relevant statistics over the TAS dashboard.
- Your code or business logic never leaves your setup environment.
- As your workload increases you can add more servers running Test-at-scale binary, which will distribute the load amongst them automatically.
- Routing: TAS platform will send the test execution jobs  to the connected self hosted environments  which are online and have enough resources to run the job.
- If the resources are insufficient or fully occupied, the jobs will remain queued on for 2.5 hour and keep checking for resource availability every 30 seconds.
- If TAS platform is unable to find any connected self-hosted binary which can execute the job, it will be marked as failed.
 
</details>

### Set up your branch to write code

We use [Github Flow](https://guides.github.com/introduction/flow/index.html), so all code changes happen through pull requests. [Learn more.](https://blog.scottlowe.org/2015/01/27/using-fork-branch-git-workflow/) 

 1. Please make sure there is an issue associated with the work that you're doing. If it doesn’t exist, [create an issue.](https://github.com/LambdaTest/test-at-scale/issues)
 2. If you're working on an issue, please comment that you are doing so to prevent duplicate work by others also.
 3. Fork the repo and create a new branch from the `dev` branch.
 4. Please name the branch as <span style="color:grey">issue-[issue-number]-[issue-name(optional)]</span> or <span style="color:grey">feature-[feature-number]–[feature-name(optional)]</span>. For example, if you are fixing Issue #205 name your branch as <span style="color:grey">issue-205 or  issue-205-selectbox-handling-changes</span>
 5. Squash your commits and refer to the issue using `Fix #<issue-no>` in the commit message, at the start.
 6. Rebase `dev` with your branch and push your changes.
 7. Raise a pull request against the staging branch of the main repository.


## **Committing code**

The repository contains two important (protected) branches.

 * main contains the code that is tested and released. 
 * dev contains recent developments under testing. This branch is set as the default branch, and all pull requests should be made against this branch.

Pull requests should be made against the <span style="color:grey">dev</span> branch. <span style="color:grey">staging</span> contains all of the new features and fixes that are under testing and ready to go out in the next release.


#### **Commit & Create Pull Requests** 

 1. Please make sure there is an issue associated with the work that you're doing. If it doesn’t exist, [create an issue](https://github.com/LambdaTest/test-at-scale/issues).
 2. Squash your commits and refer to the issue using `Fix #<issue-no>` in the commit message, at the start.
 3. Rebase `dev` with your branch and push your changes.
 4. Once you are confident in your code changes, create a pull request in your fork to the `dev` branch in the LambdaTest/test-at-scale base repository.
 5. Link the issue of the base repository in your Pull request description. [Guide](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue)
 6. Fill out the [Pull Request Template](./.github/pull_request_template.md) completely within the body of the PR. If you feel some areas are not relevant add `N/A` but don’t delete those sections.


####  **Commit messages**

- The first line should be a summary of the changes, not exceeding 50
  characters, followed by an optional body that has more details about the
  changes. Refer to [this link](https://github.com/erlang/otp/wiki/writing-good-commit-messages)
  for more information on writing good commit messages.

- Don't add a period/dot (.) at the end of the summary line.
