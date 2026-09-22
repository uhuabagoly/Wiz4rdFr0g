param([Parameter(Mandatory=$true)][string]$Campaign,[string]$Ref='codex/physical-validation-20260922',[switch]$Download,[long[]]$RefreshRuns)
$ErrorActionPreference='Stop'
if($Campaign -notmatch '^[A-Za-z0-9_-]+$'){throw 'Invalid campaign'}
$root=Join-Path 'release/physical-campaigns' $Campaign
$expected=@((Get-Content (Join-Path $root 'catalog/matrix.json') -Raw|ConvertFrom-Json).rows)
$shards=@(Get-ChildItem $root -Filter 'shards-attempt-*.json'|ForEach-Object{Get-Content $_.FullName -Raw|ConvertFrom-Json})
$allRuns=@();$page=1
while($true){
 $raw=& "$PSScriptRoot/github-actions.ps1" -Endpoint ('actions/runs?branch='+[uri]::EscapeDataString($Ref)+"&per_page=100&page=$page")
 $listing=$raw|ConvertFrom-Json;$allRuns+=@($listing.workflow_runs)
 if($listing.workflow_runs.Count -lt 100){break};$page++
}
$matched=@();$records=@();$failures=@()
foreach($shard in $shards){
 $titlePart='| '+$shard.shard_id+' | attempt '+$shard.attempt
 $run=$allRuns|Where-Object{$_.display_title.EndsWith($titlePart,[StringComparison]::Ordinal)}|Sort-Object created_at -Descending|Select-Object -First 1
 if(-not $run){continue}
 $matched+=,[pscustomobject]@{shard=$shard.shard_id;attempt=$shard.attempt;run_id=$run.id;url=$run.html_url;status=$run.status;conclusion=$run.conclusion;commit=$run.head_sha}
 $folder=Join-Path $root "runs/$($run.id)"
 $refresh=$Download -and (-not $RefreshRuns.Count -or $run.id -in $RefreshRuns)
 New-Item -ItemType Directory -Force $folder|Out-Null
 $jobsPath=Join-Path $folder 'ci-jobs.json'
 if($refresh -or -not(Test-Path $jobsPath)){
  $jobRaw=& "$PSScriptRoot/github-actions.ps1" -Endpoint "actions/runs/$($run.id)/jobs?per_page=100"
  $jobRaw|Set-Content $jobsPath
 }
 $jobs=@((Get-Content $jobsPath -Raw|ConvertFrom-Json).jobs)
 if($refresh -and -not(Test-Path (Join-Path $folder 'download-complete'))){
  $artifactRaw=& "$PSScriptRoot/github-actions.ps1" -Endpoint "actions/runs/$($run.id)/artifacts?per_page=100"
  $artifacts=@(($artifactRaw|ConvertFrom-Json).artifacts)
  $reportArtifacts=@($artifacts|Where-Object{$_.name -like 'pilot-report-*' -or $_.name -like 'linux-physical-report-*'})
  if($reportArtifacts.Count -eq 0){$reportArtifacts=@($artifacts|Where-Object{$_.name -like 'wf-package-*' -or $_.name -like 'linux-physical-result-*'})}
  foreach($artifact in $reportArtifacts){
   $zip=Join-Path $folder "$($artifact.id).zip"
   if(-not(Test-Path $zip)){& "$PSScriptRoot/github-actions.ps1" -Endpoint "actions/artifacts/$($artifact.id)/zip" -OutFile $zip}
   if($artifact.digest -and ('sha256:'+(Get-FileHash $zip -Algorithm SHA256).Hash.ToLowerInvariant()) -ne $artifact.digest){throw 'Artifact archive hash mismatch'}
   $destination=Join-Path $folder "$($artifact.id)"
   Expand-Archive -LiteralPath $zip -DestinationPath $destination -Force
  }
  if($run.status -eq 'completed' -and $reportArtifacts.Count -gt 0){Set-Content (Join-Path $folder 'download-complete') ([DateTime]::UtcNow.ToString('o'))}
 }
 foreach($file in @(Get-ChildItem $folder -Recurse -Filter '*.json')){
  $data=Get-Content $file.FullName -Raw|ConvertFrom-Json
  if($file.Name -eq 'physical-validation.json'){$data=@($data)}elseif($data.catalog_app_name -or ($data.platform -eq 'linux' -and $data.app_name)){$data=@($data)}else{continue}
  foreach($r in $data){
   if($r.final_status -eq 'MISSING'){continue}
   $index=$r.catalog_index
   if($null -eq $index -or $index -notin $shard.indexes){continue}
   $platform=$shard.platform;$apps=@($expected|Where-Object platform -eq $platform)
   if($index -lt 0 -or $index -ge $apps.Count){throw 'Out of range evidence index'}
   $want=$apps[$index];$name=if($platform -eq 'windows'){$r.catalog_app_name}else{$r.app_name}
   if($name -cne $want.app_name -or $r.git_commit -ne $run.head_sha){throw 'Evidence app identity or commit mismatch'}
   $actualID=if($platform -eq 'linux'){$r.app_id}else{$r.catalog_app_id}
   if(-not $want.evidence_app_id -or $actualID -cne $want.evidence_app_id){throw 'Evidence app ID mismatch'}
   $expectedRun=if($platform -eq 'linux'){[string]$run.id}else{"github-$($run.id)-$($run.run_attempt)"}
   $actualAttempt=if($platform -eq 'linux'){$r.campaign_attempt}else{$r.attempt}
   if($r.test_run_id -cne $expectedRun -or [string]$actualAttempt -cne [string]$shard.attempt -or $r.campaign_id -cne $shard.shard_id){throw 'Evidence run/campaign/attempt mismatch'}
   $row=[ordered]@{app_id=$want.app_id;app_name=$want.app_name;platform=$platform;catalog_index=$index;attempt=$shard.attempt;test_run_id=$run.id;git_commit=$r.git_commit;version=$r.resolved_version;download_test='FAIL';install_test='FAIL';detection_test='FAIL';uninstall_test='FAIL';post_uninstall_test='FAIL';final_status='FAIL';failure_reason=$r.failure_reason;evidence=$file.FullName;ci_run=$run.html_url;source_status=$r.final_status}
   if($platform -eq 'linux'){
    if($r.download_real -and $r.file_validation -and $r.download_http_status -eq 200 -and $r.downloaded_bytes -gt 0 -and $r.sha256 -cmatch '^[a-f0-9]{64}$'){$row.download_test='PASS'}
    if($r.install_real -and $r.install_exit_code -eq 0){$row.install_test='PASS'}
    if($r.independent_detection -and $r.wiz4rd_detection){$row.detection_test='PASS'}
    if($r.uninstall_real -and $r.uninstall_exit_code -eq 0){$row.uninstall_test='PASS'}
    if($r.independent_removed_detection -and $r.wiz4rd_removed_detection){$row.post_uninstall_test='PASS'}
   }else{
    $row.failure_reason=if($r.failure){$r.failure}else{$r.skip_reason}
    $d=$r.download_proof;$f=$r.filesystem_proof
    if($r.download_ok -and $d.file_validation -and $d.download_http_status -eq 200 -and $d.downloaded_bytes -gt 0 -and $d.sha256 -cmatch '^[a-f0-9]{64}$' -and $d.sha256 -ceq $d.expected_sha256){$row.download_test='PASS'}
    if($r.install_ok -and $r.install_exit_code -eq 0){$row.install_test='PASS'}
    if($r.install_verified -and $r.independent_installed.state -eq 'present' -and $f.registry_present -and $f.binaries_present){$row.detection_test='PASS'}
    if($r.uninstall_ok -and $r.uninstall_exit_code -eq 0){$row.uninstall_test='PASS'}
    if($r.uninstall_verified -and $r.independent_removed.state -eq 'absent' -and $f.registry_removed -and $f.binaries_removed){$row.post_uninstall_test='PASS'}
   }
   if($r.final_status -eq 'FULL_PASS' -and $row.download_test -eq 'PASS' -and $row.install_test -eq 'PASS' -and $row.detection_test -eq 'PASS' -and $row.uninstall_test -eq 'PASS' -and $row.post_uninstall_test -eq 'PASS'){$row.final_status='FULL_PASS'}
   $jobName=if($platform -eq 'linux'){"lifecycle ($index)"}else{"package-test ($index)"}
   $job=@($jobs|Where-Object name -CEQ $jobName)
   if($row.final_status -eq 'FULL_PASS' -and ($job.Count -ne 1 -or $job[0].conclusion -ne 'success')){
    $row.final_status='FAIL';$row.failure_reason='Matching physical CI job did not complete successfully; artifact validation is not established'
   }
   $row.ci_job_id=if($job.Count -eq 1){$job[0].id}else{$null}
   $row.os_version=if($platform -eq 'linux'){$r.os_version}else{$r.environment.windows_version}
   $row.architecture=if($platform -eq 'linux'){$r.architecture}else{$r.environment.architecture}
   $row.expected_version=if($platform -eq 'linux'){$r.expected_version}else{$r.resolved_version}
   $row.resolved_download_url=if($platform -eq 'linux'){$r.resolved_download_url}else{$d.resolved_download_url}
   $row.download_http_status=if($platform -eq 'linux'){$r.download_http_status}else{$d.download_http_status}
   $row.downloaded_bytes=if($platform -eq 'linux'){$r.downloaded_bytes}else{$d.downloaded_bytes}
   $row.sha256=if($platform -eq 'linux'){$r.sha256}else{$d.sha256}
   $row.file_validation=if($platform -eq 'linux'){$r.file_validation}else{$d.file_validation}
   $row.pre_install_detection=if($platform -eq 'linux'){$r.pre_install_detection}else{$r.independent_before.state}
   $row.install_command_type=if($platform -eq 'linux'){$r.install_command_type}else{'winget'}
   $row.install_exit_code=$r.install_exit_code
   $row.physical_started=if($platform -eq 'windows'){$r.download_exit_code -ne -999}else{$r.download_http_status -gt 0 -or $r.downloaded_bytes -gt 0 -or $null -ne $r.install_exit_code}
   $row.uninstall_mechanism=if($platform -eq 'linux'){$r.uninstall_mechanism}else{$r.detected_after_install.uninstall_strategy}
   $row.uninstall_exit_code=$r.uninstall_exit_code
   $row.independent_installed_detection=if($platform -eq 'linux'){$r.independent_detection}else{$r.independent_installed.state}
   $row.wiz4rd_installed_detection=if($platform -eq 'linux'){$r.wiz4rd_detection}else{$r.install_verified}
   $row.independent_removed_detection=if($platform -eq 'linux'){$r.independent_removed_detection}else{$r.independent_removed.state}
   $row.wiz4rd_removed_detection=if($platform -eq 'linux'){$r.wiz4rd_removed_detection}else{$r.uninstall_verified}
   $row.started_at=$r.started_at;$row.finished_at=$r.finished_at
   if($platform -eq 'linux' -and $r.provider -eq 'apt-get' -and $r.resolved_version -match 'snap'){
    $row.final_status='FAIL';$row.detection_test='FAIL';$row.post_uninstall_test='FAIL';$row.failure_reason='Transitional Snap package evidence does not establish the actual application lifecycle'
   }
   if($row.final_status -ne 'FULL_PASS' -and -not $row.failure_reason){$row.failure_reason='Required physical proof is incomplete'}
   $records+=,[pscustomobject]$row
  }
 }
}
$output=@()
foreach($app in $expected){
 $matches=@($records|Where-Object app_id -eq $app.app_id|Sort-Object @{Expression='attempt';Descending=$true},@{Expression='test_run_id';Descending=$true})
 if($matches.Count){$output+=,$matches[0]}else{$output+=,[pscustomobject]@{app_id=$app.app_id;app_name=$app.app_name;platform=$app.platform;catalog_index=([array]@($expected|Where-Object platform -eq $app.platform)).IndexOf($app);attempt=$null;test_run_id=$null;git_commit=$null;version=$null;download_test='MISSING';install_test='MISSING';detection_test='MISSING';uninstall_test='MISSING';post_uninstall_test='MISSING';final_status='MISSING';failure_reason='No collected physical evidence';evidence=$null;ci_run=$null;source_status=$null}}
}
$matched|ConvertTo-Json -Depth 8|Set-Content (Join-Path $root 'runs.json')
 $allColumns=@($output|ForEach-Object {$_.PSObject.Properties.Name}|Select-Object -Unique)
 $output=@($output|Select-Object $allColumns)
ConvertTo-Json -InputObject $output -Depth 10|Set-Content (Join-Path $root 'physical-validation.json')
$output|Export-Csv (Join-Path $root 'physical-validation.csv') -NoTypeInformation -Encoding utf8
$summary=@();foreach($platform in @('windows','linux')){$rows=@($output|Where-Object platform -eq $platform);$summary+=,[pscustomobject]@{platform=$platform;supported_apps=$rows.Count;download_pass=@($rows|Where-Object download_test -eq 'PASS').Count;install_pass=@($rows|Where-Object install_test -eq 'PASS').Count;detection_pass=@($rows|Where-Object detection_test -eq 'PASS').Count;uninstall_pass=@($rows|Where-Object uninstall_test -eq 'PASS').Count;full_lifecycle_pass=@($rows|Where-Object final_status -eq 'FULL_PASS').Count;fail=@($rows|Where-Object final_status -eq 'FAIL').Count;missing=@($rows|Where-Object final_status -eq 'MISSING').Count}}
$executedIDs=@(($records|Where-Object physical_started).app_id|Sort-Object -Unique)
$gate=[ordered]@{generated_at=[DateTime]::UtcNow.ToString('o');platforms=$summary;expected_app_ids=@($expected.app_id);processed_app_ids=@($records.app_id|Sort-Object -Unique);executed_app_ids=$executedIDs;fully_validated_app_ids=@(($output|Where-Object final_status -eq 'FULL_PASS').app_id);not_physically_started_app_ids=@(($expected|Where-Object {$_.app_id -notin $executedIDs}).app_id);missing_app_ids=@(($output|Where-Object final_status -eq 'MISSING').app_id);result=if(@($output|Where-Object final_status -ne 'FULL_PASS').Count){'RELEASE_GATE_FAIL'}else{'RELEASE_GATE_PASS'}}
$gate|ConvertTo-Json -Depth 8|Set-Content (Join-Path $root 'release-gate.json')
$summary|Format-Table
$matched|Group-Object status,conclusion|Select-Object Name,Count
