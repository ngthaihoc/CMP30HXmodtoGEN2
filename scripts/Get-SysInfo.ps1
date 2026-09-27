$ErrorActionPreference = "SilentlyContinue"

$cs = Get-CimInstance Win32_ComputerSystem
$bb = Get-CimInstance Win32_BaseBoard
$bios = Get-CimInstance Win32_BIOS
$batt = Get-CimInstance Win32_Battery

$sysManu = $cs.Manufacturer
$sysModel = $cs.Model
$mbManu = $bb.Manufacturer
$mbProd = $bb.Product
$mbBios = $bios.SMBIOSBIOSVersion

$isLaptop = 0
if ($cs.PCSystemType -eq 2 -or $batt) {
    $isLaptop = 1
}

Write-Output "SYS_MANU=$sysManu"
Write-Output "SYS_MODEL=$sysModel"
Write-Output "MB_MANU=$mbManu"
Write-Output "MB_PROD=$mbProd"
Write-Output "MB_BIOS=$mbBios"
Write-Output "IS_LAPTOP=$isLaptop"
