param(
 [Parameter(Mandatory=$true)][string]$Campaign,
 [string]$Ref='codex/physical-validation-20260922',
 [ValidateRange(1,128)][int]$ShardSize=32,
 [ValidateRange(1,1000)][int]$Attempt=1,
 [string]$FailedOnlyReport,
 [switch]$Dispatch
)
$ErrorActionPreference='Stop'
if($Campaign -notmatch '^[A-Za-z0-9_-]+$'){throw 'Invalid campaign name'}
$root=Join-Path 'release/physical-campaigns' $Campaign
New-Item -ItemType Directory -Force $root | Out-Null
& go run ./cmd/catalog-matrix -out (Join-Path $root 'catalog')
if($LASTEXITCODE -ne 0){throw 'Catalog derivation failed'}
$catalog=Get-Content (Join-Path $root 'catalog/matrix.json') -Raw|ConvertFrom-Json
$prior=@()
if($FailedOnlyReport){$prior=@(Get-Content -LiteralPath $FailedOnlyReport -Raw|ConvertFrom-Json)}
$shards=@()
foreach($platform in @('windows','linux')){
 $apps=@($catalog.rows|Where-Object platform -eq $platform)
 $indexes=@(0..($apps.Count-1))
 if($FailedOnlyReport){
  $passed=@{};foreach($row in $prior){if($row.platform -eq $platform -and $row.final_status -eq 'FULL_PASS'){$passed[$row.app_id]=$true}}
  $indexes=@($indexes|Where-Object {-not $passed.ContainsKey($apps[$_].app_id)})
 }
 for($start=0;$start -lt $indexes.Count;$start+=$ShardSize){
  $end=[Math]::Min($start+$ShardSize,$indexes.Count)
  $selected=@($indexes[$start..($end-1)])
  $number=[int]($start/$ShardSize)+1
  $id='{0}-{1}-{2:D4}' -f $Campaign,$platform,$number
  $workflow=if($platform -eq 'windows'){'windows-vm-package-test.yml'}else{'linux-physical.yml'}
  $request=@{ref=$Ref;inputs=@{indexes=(ConvertTo-Json -InputObject $selected -Compress);campaign_id=$id;attempt="$Attempt"}}
  $body=Join-Path $root "$id-attempt-$Attempt-request.json"
  $request|ConvertTo-Json -Depth 6|Set-Content $body
  $record=[ordered]@{shard_id=$id;platform=$platform;attempt=$Attempt;workflow=$workflow;indexes=$selected;app_ids=@($selected|ForEach-Object{$apps[$_].app_id});request_file=$body;dispatch_status='NOT_DISPATCHED';dispatched_at=$null}
  $receipt=Join-Path $root "$id-attempt-$Attempt-receipt.json"
  if(Test-Path $receipt){$existing=Get-Content $receipt -Raw|ConvertFrom-Json;if($existing.dispatch_status -eq 'DISPATCHED'){$record=$existing}}
  if($Dispatch -and $record.dispatch_status -ne 'DISPATCHED'){
   & "$PSScriptRoot/github-actions.ps1" -Endpoint "actions/workflows/$workflow/dispatches" -Method POST -BodyFile $body
   $record.dispatch_status='DISPATCHED';$record.dispatched_at=[DateTime]::UtcNow.ToString('o')
   $record|ConvertTo-Json -Depth 8|Set-Content $receipt
  }
  $shards+=,[pscustomobject]$record
  $shards|ConvertTo-Json -Depth 8|Set-Content (Join-Path $root "shards-attempt-$Attempt.json")
 }
}
$shards|Select-Object shard_id,dispatch_status,@{Name='apps';Expression={$_.indexes.Count}}
