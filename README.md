# sc2replay-discord

A Discord bot that responds with a summary of `.SC2Replay` files posted in a text channel.

[Invite to your Discord server.](https://discord.com/api/oauth2/authorize?client_id=909174200068079716&permissions=2048&scope=bot)

## Deploy to AWS

```
# Put your bot's token to SSM Parameter Store.
$ aws ssm put-parameter \
  --name "/$APP_STACK_NAME/DISCORD_BOT_TOKEN" \
  --type "SecureString" \
  --value $DISCORD_BOT_TOKEN

# This deploys CodePipeline and CI resources. Then the pipeline deploys app.
$ aws cloudformation deploy \
  --stack-name "sc2replay-discord-ci" \
  --template-file "infra/ci.cfn.json" \
  --capabilities "CAPABILITY_IAM" \
  --role-arn $CFN_ROLE_ARN \
  --parameter-overrides AppStackName=$APP_STACK_NAME GithubConnectionArn=$GITHUB_CONNECTION_ARN FullRepositoryId=$FULL_REPOSITORY_ID BranchName=$BRANCH_NAME SlackChannelId=$SLACK_CHANNEL_ID SlackWorkspaceId=$SLACK_WORKSPACE_ID
```
