# go-tfe-cleanup

I often find Terraform Cloud workspaces are "stuck" because:

```
Note: Runs initiated in your workspace as the result of a run trigger connection to a source workspace will not auto-apply, regardless of your auto-apply setting selection. You will need to manually apply these runs.
```
--[source](https://www.terraform.io/cloud-docs/workspaces/settings/run-triggers#creating-a-run-trigger)

## Usage

```
$ go run main.go -org "myorg" -search "dev-"
2022/05/13 16:28:25 run_id=run-e3wMieRxtxmVXx,workspace_name=dev-mod-main,action=skip
2022/05/13 16:28:25 run_id=run-LjwRn76Qcs5dyq,workspace_name=dev-mod-main,action=cancel
2022/05/13 16:28:25 run_id=run-FzzCtL1GHBhP4c,workspace_name=dev-mod-main,action=cancel
2022/05/13 16:28:25 run_id=run-xJ1MrjuSHb5UZo,workspace_name=dev-mod-main,action=discard
```

The latest run is skipped as it will run once the other runs are discarded or canceled. If the latest run is the stuck run, it will apply based on the `auto-apply` setting of the workspace.