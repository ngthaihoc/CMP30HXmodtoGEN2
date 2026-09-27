Restart-Service -Name 'NVDisplay.ContainerLocalSystem' -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1
$s = Get-Service -Name 'NVDisplay.ContainerLocalSystem' -ErrorAction SilentlyContinue
if ($s -and $s.Status -ne 'Running') {
    Start-Service -Name 'NVDisplay.ContainerLocalSystem' -ErrorAction SilentlyContinue
}
