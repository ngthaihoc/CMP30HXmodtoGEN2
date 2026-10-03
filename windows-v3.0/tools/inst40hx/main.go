// 40HX Σ╕ÇΘö«σ«ëΦúàσ╖Ñσà╖ v3.0.0 (CMP 40HX Windows Unlock Installer)
// σèƒΦâ╜:
//
//	(Θ╗ÿΦ«ñ) σ«ëΦúà: GSP σÉ»τö¿(EnableGpuFirmware=1) + ESP σÅîΦ╖»Θâ¿τ╜▓ 40HXUNLK.EFI (V70)
//	      + BootOrder τ╜«Θí╢ + Θ⌐▒σè¿ + Gen2 Φç¬σÉ»σè¿
//	-gen2        τ½ïσì│µëºΦíî Gen2 ΦºúΘöü(Σ╛¢τÖ╗σ╜òΦç¬σÉ»σè¿Φ░âτö¿, σ╣éτ¡ë)
//	-uninstall   σì╕Φ╜╜(τº╗ΘÖñσÉ»σè¿Θí╣/RunΘö«/Θ⌐▒σè¿µ£ìσèí/EnableGpuFirmware)
//	-status      τè╢µÇüµúÇµƒÑ
//
// Φ╡äµ║É embed (v2.5): 40HXUNLK.EFI (V70 ΦºúΘöüτëê) / ThrottleStop.sys / WinRing0x64.sys
// v2.6.0 σà│Θö«Σ┐«σñì(τñ╛σî║ #2/#5/#6/#7 + v2.4.5 µù╢Σ╗úµÄÆΘÜ£τ╗ôΦ«║):
//  1. EFI Θâ¿τ╜▓σñ▒Φ┤ÑΣ╕ìσåìΣ╕¡µ¡óσ«ëΦúà ΓÇö Legacy/MBR(µùá ESP)σÅ¬Φ╖│Φ┐ç EFI Σ╕ñµ¡Ñ, Gen2 Σ╗╗σèí
//     τàºσ╕╕µ│¿σåî(µ¡ñσëì [5/8] τ¢┤µÄÑ return, µÿ»"ΦúàΣ║åΘ⌐▒σè¿σ╝Çµ£║σì┤Σ╕ìΦ╖æ Gen2"τÜäτ╗ƒΣ╕Çµá╣σ¢á)
//  2. σ╝òσ»╝µ¿íσ╝ÅµúÇµ╡ï(GetFirmwareType): Legacy ΓåÆ σ╝╣τ¬ùτ╗Ö mbr2gpt µùáµìƒΦ╜¼µìóσ«îµò┤µîçσ╝ò
//  3. Φ«íσêÆΣ╗╗σèíσê¢σ╗║σÉÄ schtasks query Σ║îµ¼íµáíΘ¬î + ΘçìΦ»ò; -task σñ▒Φ┤ÑΣ╗ÑΘ¥₧Θ¢╢τáüΘÇÇσç║
//     (σæ╜Σ╗ñΦíîΦ░âτö¿µù╢τÜä errorlevel µúÇµƒÑΣ╗Äµ¡╗Σ╗úτáüσÅÿΣ╕║µ£ëµòê)
//  4. Φç¬σè¿σà│Θù¡σ┐½ΘÇƒσÉ»σè¿(µ╖╖σÉêΣ╝æτ£á)Σ╕Ä PCIe Θô╛Φ╖»τ£üτö╡(ASPM) ΓÇö σëìΦÇàΘü┐σàì"σà│µ£║σåìσ╝Ç
//     Σ╕ìΦ╡░σ«îµò┤ UEFI σ╝òσ»╝", σÉÄΦÇàσçÅσ░æτ⌐║Θù▓ΘÖìσê░ Gen1 Φó½Φ»»Φ»╗Σ╕║ΦºúΘöüσñ▒Φ┤Ñ
//  5. Gen2 µá╕σ┐âσó₧σ╝║: LNKCTL2 Φ»╗µö╣σåÖ(Σ╕ìµ╕àΘ½ÿΣ╜ì) + root/GPU Σ║ñµ¢┐ΘçìΦ«¡µ£ÇσñÜ 4 Φ╜« +
//     Σ╗Ñ TLS τ¢«µáçΘÇƒτÄçσêñµêÉΦ┤Ñ(τ⌐║Θù▓τ£üτö╡ΘÖìΘÇƒ Gen1 Σ╕ìσåìΦ»»µèÑσñ▒Φ┤Ñ)
//
// v2.6.0 σà│Θö«σèáσ¢║(Φç¬σÉ»σè¿ΘÇÜΘüôΦ«╛Φ«íΣ╕Äσ╣╢σÅæσ«ëσà¿, σ¢₧σ║ö"σñÜΦç¬σÉ»σè¿Φ╖»σ╛äµÇòσç║Θù«Θóÿ"):
//  1. Gen2 σìòσ«₧Σ╛ïσåàµá╕Σ║ÆµûÑΣ╜ô(Global\40HXGen2SingleInstance): SYSTEM Σ╗╗σèí / Run Θö« /
//     µëïσè¿ -gen2 σì│Σ╜┐σ╣╢σÅæΦºªσÅæ, Σ╣ƒΣ╗àΣ╕ÇΣ╕¬Φ┐¢τ¿ïΦ┐¢σàÑ"σèáΦ╜╜-σì╕Φ╜╜ BYOVD Θ⌐▒σè¿ + µèó BAR0"
//     Σ╕┤τòîσî║, µ¥£τ╗¥σÅîΦ┐¢τ¿ïΣ║ëτö¿Θ⌐▒σè¿µ£ìσèíσÉìΣ╕ÄΘô╛Φ╖»σ»äσ¡ÿσÖ¿σ»╝Φç┤τÜäτè╢µÇüΘöÖΣ╣▒
//  2. Φç¬σÉ»σè¿ΘÇÜΘüôµö╢µò¢Σ╕║"Σ╕ñΦ╖»Σ║ÆµûÑΣ╕▓Φíî": Run Θö«τÖ╗σ╜òτ₧¼Θù┤σàêΦ»ò(σÅ»Φâ╜ GPU µ£¬σ░▒τ╗¬ΦÇîσñ▒Φ┤Ñ,
//     Θ¥ÖΘ╗ÿΣ║ñµ¥â), SYSTEM Σ╗╗σèíσ╗╢Φ┐ƒ 30s σåìτí«Φ«ñ; σà╢Σ╜Ö 13 τ▒╗Φ╖»σ╛ä(HKCU/HKLM Run Σ╣ïσñû)
//     σ¥çΦ┐ÉΦíîΣ║Äτö¿µê╖µÇüπÇüµùáµ│ò sc start σåàµá╕Θ⌐▒σè¿, µòàΣ╕ìΘççτö¿(Φ»ªΦºüΦ«╛Φ«íµûçµíú)
//  3. σ«ÜΣ╜ì 40HX σñ▒Φ┤ÑΘçìΦ»òµ£ÇσñÜ 3 µ¼í(Θù┤ΘÜö 2s), σ«╣σ┐ìµàóΘÇƒ GPU σê¥σºïσîûσ»╝Φç┤τÜäσüçσñ▒Φ┤Ñ
//
// v2.4 σà│Θö«σÅÿµ¢┤(τñ╛σî║σà╝σ«╣):
//  1. embed EFI σ¢₧σê░ V70 σÄƒτëê (793d765e, τö¿µê╖σ«₧µ╡ïΦºúΘöüµêÉσèƒ) ΓÇö v2.1/v2.2 τ▓╛τ«Çτëêσñ▒Φ┤ÑµòÖΦ«¡
//  2. ESP σÅîΦ╖»Θâ¿τ╜▓: \EFI\40HX\40HXUNLK.EFI (BCD Σ╕╗Φ╖»σ╛ä)
//     + \EFI\Boot\bootx64.efi (UEFI µáçσçå fallback, σÄƒµûçΣ╗╢σñçΣ╗╜ .40hx.bak)
//     Φºúσå│Θâ¿σêåΣ╕╗µ¥┐Σ╕ìΦ«ñΘ¥₧µáçσçå EFI Φ╖»σ╛ä/σ┐╜τòÑ BCD displayorder σ»╝Φç┤"Φúàσ«îΘçìσÉ»µ▓íσÅìσ║ö"
//  3. BootOrder σåÖσàÑσÉÄΣ╗Äσ¢║Σ╗╢Φ»╗σ¢₧Θ¬îΦ»ü, Σ╕ìσ£¿ΘªûΣ╜ìµù╢µÿÄτí«σ╝╣τ¬ùµÅÉτñ║ BIOS µëïσè¿τ╜«Θí╢
//  4. σà│Θö« BIOS µôìΣ╜£σà¿Θâ¿Φ┐¢µ╢êµü»µíå (τñ╛σî║τö¿µê╖Σ╕ìτ£ï README/µùÑσ┐ù)
//
// v2.3 σà│Θö«: EnableGpuFirmware=1 σÉ»τö¿ GSP ΓÇö 40HX Θ╗ÿΦ«ñ GSP σà│(CPU-RM µ¿íσ╝Å)µù╢,
//
//	EFI ΦºúΘöüσÉÄ nvlddmkm µïÆτ╗¥ SEC2 τè╢µÇü -> Code43 Θ╗æσ▒Å; GSP-RM µ¿íσ╝ÅΦâ╜µÄÑσÅùΦºúΘöü.
package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"40hxcore"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed embed/*
var embedded embed.FS

const (
	gpuVenDev = "VEN_10DE&DEV_1F0B"
	efiDir    = "\\EFI\\40HX"
	efiFile   = "40HXUNLK.EFI"
	bootDesc  = "40HX Unlock"
	// v2.4: UEFI µáçσçåσ¢₧ΘÇÇΦ╖»σ╛ä (σ¢║Σ╗╢ BootOrder σà¿Θâ¿µùáµòê/µ£¬τ¡╛σÉìµù╢Φç¬σè¿σ░¥Φ»òµ¡ñΦ╖»σ╛ä;
	// Φºúσå│Θâ¿σêåΣ╕╗µ¥┐σ┐╜τòÑ BCD displayorder / Σ╕ìΦ«ñΘ¥₧µáçσçå \EFI\40HX τ¢«σ╜ò)
	efiStdDir = "\\EFI\\Boot"
	efiStdF   = "bootx64.efi"
	efiBakExt = ".40hx.bak" // bootx64.efi.40hx.bak σÄƒµûçΣ╗╢σñçΣ╗╜
	// v2.3: GSP σÉ»τö¿µ│¿σåîΦí¿ (EnableGpuFirmware=1) ΓÇö ΦºúΘöüΣ╕ìΘ╗æσ▒ÅτÜäσà│Θö«!
	// 40HX τÜäµÿ╛τñ║ΘÇéΘàìσÖ¿ Class σ¡ÉΘö« (0001 = 40HX; σñÜσìíµù╢Θ£Çµîë AdapterString µë╛)
	gpuClassPath  = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	gpuClassGUID  = `{4d36e968-e325-11ce-bfc1-08002be10318}` // Driver σÇ╝σÅìµƒÑτö¿
	gpuEnableFw   = "EnableGpuFirmware"
	gpuAdapterStr = "HardwareInformation.AdapterString"
	gpuAdapter40  = "CMP 40HX"
	// v2.4.6: Gen2 τÜä SYSTEM Φ«íσêÆΣ╗╗σèíσÉì(σì╕Φ╜╜µù╢µîëσÉìσ¡ùσêáΘÖñ)
	gen2TaskName = "40HX PCIe Gen2 Bring-up"
	// v2.6.0: Gen2 σñ▒Φ┤ÑσÉÄτÜäΦç¬σè¿ΘçìΦ»òΣ╗╗σèí(Σ╕Çµ¼íµÇº, µêÉσèƒσì│σêá, σì╕Φ╜╜Θô╛µîëσÉìµ╕àτÉå)
	gen2RetryTask = "40HXGen2Retry"
)

func main() {
	hxcore.DriverFileProvider = func(filename string) ([]byte, error) {
		return embedded.ReadFile("embed/" + filename)
	}
	// GUI µùáτ¬ùσÅúτëê(v1.1): Φ╛ôσç║σà¿Θâ¿Θò£σâÅσê░µùÑσ┐ù(Θ╗ÿΦ«ñ %TEMP%\40HX_installer.log, σÅ» -log µîçσ«Ü)
	setupLog("40HX_installer.log")
	// v3.0: Mß║╖c ─æß╗ïnh khß╗ƒi chß║íy Modern Web Control Center khi nhß║Ñp ─æ├║p hoß║╖c UAC -elevated
	if hasArg("-gui-classic") {
		runGUI()
		return
	}
	if len(os.Args) <= 1 || (len(os.Args) == 2 && os.Args[1] == "-elevated") || hasArg("-ui") || hasArg("-web") || hasArg("-gui") {
		runWebGUI()
		return
	}
	// install/-uninstall Θ£Çτ«íτÉåσæÿ: Θ¥₧µÅÉσìçµù╢Φç¬σè¿ ShellExecute runas σ╝╣ UAC ΘçìσÉ»
	needAdmin := true
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-gen2", "-gen3", "-force-root-gen2", "-force-root-gen3", "-gen2-30hx", "-gen3-30hx", "-probe-30hx", "-gspensure", "-status", "-h", "-help", "--help":
			needAdmin = false
		}
		// -task Θ£Çτ«íτÉåσæÿ(GUI σÅîσç╗Φç¬σè¿ UAC; gen2/status τ¡ëσÅ¬Φ»╗µêû SYSTEM Σ╗╗σèíΦ░âτö¿µùáΘ£Ç)
		if os.Args[1] == "-task" || os.Args[1] == "-probe-30hx" || os.Args[1] == "-gen2-30hx" || os.Args[1] == "-gen3-30hx" {
			needAdmin = true
		}
	}
	if needAdmin && !isAdmin() {
		if hasArg("-elevated") {
			// σ╖▓µÅÉµ¥âΦ┐çΣ╕Çµ¼íΣ╗ìσñ▒Φ┤Ñ(σªéΘ¥ÖΘ╗ÿµÅÉµ¥âτ¡ûτòÑΣ╕ïσÅùΘÖÉtoken) -> τªüµ¡óσåìσ╛¬τÄ», τ¢┤µÄÑµèÑΘöÖ
			msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "N├óng quyß╗ün thß║Ñt bß║íi: T├ái khoß║ún hiß╗çn tß║íi kh├┤ng c├│ quyß╗ün Quß║ún trß╗ï vi├¬n (Administrator).\nVui l├▓ng nhß║Ñp chuß╗Öt phß║úi v├áo ß╗⌐ng dß╗Ñng -> Chß╗ìn 'Run as administrator'.", mbIconError)
			return
		}
		selfElevate()
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-gen2", "-gen3", "-force-root-gen2", "-force-root-gen3", "-gen2-30hx", "-gen3-30hx":
			gen2Succeeded = false
			gen2Main()
			if !gen2Succeeded {
				os.Exit(1)
			}
			// v3.0.1: σ╕╕Θ⌐╗σ«êµèñ ΓÇö τö▒τÖ╗σ╜òΣ╗╗σèíσ╕ª -guard σÉ»σè¿; Θ⌐▒σè¿Σ┐¥τòÖσ╣╢µ»ÅσêåΘÆƒΦç¬µƒÑ Gen2
			if hasArg("-guard") && hxcore.DriverStrategy() == hxcore.DriverStrategyResident {
				residentGuard()
			}
			return
		case "-probe-30hx":
			probe30HX()
			return
		case "-gspensure":
			gspEnsureMain()
			return
		case "-reset":
			resetPnpMain()
			return
		case "-uninstall":
			uninstall()
			return
		case "-status":
			status()
			return
		case "-task":
			// Σ╗àµ│¿σåî Gen2 τÖ╗σ╜òΦç¬σÉ»Σ╗╗σèí(Σ╛¢ -task µ¿íσ╝ÅΦ░âτö¿;
			// τö▒ Go µ₧äΘÇá /TR σ╝òσÅ╖, Θü┐σàì bat σåàσ╡îσ╝òσÅ╖Φºúµ₧Éσç║ΘöÖ/Θù¬ΘÇÇ)
			regTaskOnly()
			return
		case "-h", "-help", "--help":
			printHelp()
			return
		}
	}
	install()
}

// regTaskOnly: σÅ¬µ│¿σåî Gen2 SYSTEM Σ╗╗σèí(Σ╕ìσ«ëΦúàΘ⌐▒σè¿/EFI/GSP)πÇé
// -task µ¿íσ╝ÅτÜäµ£ÇσÉÄΣ╕Çµ¡ÑΦ░âτö¿µ£¼µ¿íσ╝Å ΓÇö Go σñäτÉåσ╝òσÅ╖πÇé
// v2.6.0: σñ▒Φ┤ÑΣ╗ÑΘ¥₧Θ¢╢τáüΘÇÇσç║ ΓÇö bat τÜä errorlevel µúÇµƒÑΣ╛¥Φ╡ûσ«â(µ¡ñσëìµüÆΣ╕║ 0, µúÇµƒÑµÿ»µ¡╗Σ╗úτáü)πÇé
func regTaskOnly() {
	if !isAdmin() {
		fmt.Println("[!] ─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy cß║ºn quyß╗ün Quß║ún trß╗ï vi├¬n (Administrator).")
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy cß║ºn quyß╗ün Quß║ún trß╗ï vi├¬n.\nVui l├▓ng chß║íy vß╗¢i quyß╗ün Administrator.", mbIconError)
		os.Exit(1)
	}
	if err := setupGen2Task(); err != nil {
		fmt.Println("[!]", err)
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "─É─âng k├╜ t├íc vß╗Ñ mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p thß║Ñt bß║íi:\n"+err.Error()+
			"\n\nVui l├▓ng ─æß║úm bß║úo chß║íy bß║▒ng quyß╗ün Administrator rß╗ôi thß╗¡ lß║íi.", mbIconError)
		os.Exit(1)
	}
	setRunKey()
	msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "T├íc vß╗Ñ tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p ─æ├ú ─æ╞░ß╗úc ─æ─âng k├╜ th├ánh c├┤ng.\nSau khi ─æ─âng nhß║¡p Windows, hß╗ç thß╗æng sß║╜ tß╗▒ ─æß╗Öng k├¡ch hoß║ít PCIe (chß║íy ß║⌐n, d├╣ng xong gß╗í driver).", mbIconInfo)
}

// selfElevate: Khß╗ƒi ─æß╗Öng lß║íi vß╗¢i quyß╗ün Admin qua UAC ShellExecute "runas"
func selfElevate() {
	hxcore.SelfElevate("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX")
}

var (
	gen2Succeeded bool
)

const (
	mbIconInfo  = hxcore.MbIconInfo
	mbIconError = hxcore.MbIconError
	mbIconWarn  = hxcore.MbIconWarn // MB_ICONWARNING: v2.6.0: EFI Φ╖│Φ┐ç/Θâ¿σêåµêÉσèƒτ¡ë"σÅ»τ╗ºτ╗¡Σ╜åΦªüµ│¿µäÅ"σ£║µÖ»
	mbYesNo     = hxcore.MbYesNo    // MB_YESNO ΓåÆ Φ┐öσ¢₧ IDYES=6 / IDNO=7
)

var (
	procCreateMutex = syscall.NewLazyDLL("kernel32.dll").NewProc("CreateMutexW")
)

func msgbox(title, text string, icon uint) {
	// -y / -silent(Φç¬σè¿σîû/Φç¬σÉ»σè¿) µù╢Σ╕ìσ╝╣µíå
	if hasArg("-y") || hasArg("-silent") {
		return
	}
	hxcore.MsgBox(title, text, icon)
}

// msgboxYesNo: µÿ»/σÉªΦ»óΘù«πÇéΦç¬σè¿µ¿íσ╝Å: -yΓåÆtrue(σà¿Φç¬σè¿τ╗ºτ╗¡), -silentΓåÆfalse(Σ╕ìµëôµë░)πÇé
func msgboxYesNo(title, text string) bool {
	if hasArg("-y") {
		return true
	}
	if hasArg("-silent") {
		return false
	}
	return hxcore.MsgBoxYesNo(title, text)
}

// setupLog: Φ╛ôσç║Θò£σâÅσê░µùÑσ┐ùµûçΣ╗╢(Θ╗ÿΦ«ñ %TEMP%/<name>, σæ╜Σ╗ñΦíî -log <file> Σ╝ÿσàê)
func setupLog(defName string) {
	p := filepath.Join(os.TempDir(), defName)
	if i := argIndex("-log"); i >= 0 && i+1 < len(os.Args) {
		p = os.Args[i+1]
	}
	if f, err := os.Create(p); err == nil {
		os.Stdout = f
		os.Stderr = f
		fmt.Fprintf(f, "==== 40HX tool %s ====\n", time.Now().Format("2006-01-02 15:04:05"))
	}
}

// AttachLogSink: v2.6.0 GUI τö¿ ΓÇö τö¿ os.Pipe µèèσÉÄτ╗¡ fmt.* Φ╛ôσç║σêåµ╡üσê░ µùÑσ┐ùµûçΣ╗╢+UIπÇé
// fmt.* µ»Åµ¼íΦ░âτö¿Φ»╗ os.Stdout σÅÿΘçÅ; Σ╜å os.Stdout µ£¼Φ║½µÿ» *os.File σà╖Σ╜ôτ▒╗σ₧ï,
// Σ╕ìΦâ╜Φ╡ï io.Writer, µòàµ¢┐µìóΣ╕║τ«íΘüôσåÖτ½», τö▒Φ»╗σìÅτ¿ïσÉîµù╢σåÖσÄƒµûçΣ╗╢Σ╕Ä GUI µùÑσ┐ùΘ¥óµ¥┐πÇé
func AttachLogSink(w io.Writer) {
	r, pw, err := os.Pipe()
	if err != nil {
		return
	}
	orig := os.Stdout // setupLog σ╗║τ½ïτÜäµùÑσ┐ùµûçΣ╗╢(µêû GUI Σ╕ïτÜäµùáµòêµÄºσê╢σÅ░σÅÑµƒä)
	os.Stdout = pw
	os.Stderr = pw
	go func() {
		defer r.Close()
		buf := make([]byte, 4096)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				orig.Write(buf[:n]) // ΦÉ╜µùÑσ┐ùµûçΣ╗╢(GUI µ¿íσ╝ÅΣ╕ïσñ▒Φ┤ÑσÅ»σ┐╜τòÑ)
				w.Write(buf[:n])    // σûé GUI µùÑσ┐ùΘ¥óµ¥┐
			}
			if rerr != nil {
				return
			}
		}
	}()
}

// lockOnce: σìòσ«₧Σ╛ïΣ║ÆµûÑ; Φ┐öσ¢₧ nil Φí¿τñ║σ╖▓µ£ëσ«₧Σ╛ïσ£¿Φ╖æ
func lockOnce(name string) func() {
	n, _ := syscall.UTF16PtrFromString(name)
	h, _, e := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return nil
	}
	if e == syscall.ERROR_ALREADY_EXISTS {
		syscall.CloseHandle(syscall.Handle(h))
		return nil
	}
	return func() { syscall.CloseHandle(syscall.Handle(h)) }
}

func hasArg(name string) bool {
	for _, a := range os.Args {
		if a == name {
			return true
		}
	}
	return false
}

func argIndex(name string) int {
	for i, a := range os.Args {
		if a == name {
			return i
		}
	}
	return -1
}

func printHelp() {
	fmt.Println("Tr├¼nh Mß╗ƒ Kho├í & K├¡ch Hoß║ít PCIe CMP 40HX / 30HX tr├¬n Windows")
	fmt.Println("  C├ích d├╣ng: 40HXInstaller.exe                  # C├ái ─æß║╖t giao diß╗çn / to├án bß╗Ö (cß║ºn Admin)")
	fmt.Println("             40HXInstaller.exe -gen2            # Mß╗ƒ kho├í Gen2 ngay lß║¡p tß╗⌐c")
	fmt.Println("             40HXInstaller.exe -gen3            # (CMP 30HX) Mß╗ƒ kho├í Gen3 ngay lß║¡p tß╗⌐c")
	fmt.Println("             40HXInstaller.exe -force-root-gen2 # (CMP 30HX) ├ëp Root Port huß║Ñn luyß╗çn lß║íi Gen2")
	fmt.Println("             40HXInstaller.exe -force-root-gen3 # (CMP 30HX) ├ëp Root Port huß║Ñn luyß╗çn lß║íi Gen3")
	fmt.Println("             40HXInstaller.exe -gen2-30hx       # (CMP 30HX) MMIO ghi ─æ├¿ + Huß║Ñn luyß╗çn lß║íi Gen2")
	fmt.Println("             40HXInstaller.exe -gen3-30hx       # (CMP 30HX) MMIO ghi ─æ├¿ + Huß║Ñn luyß╗çn lß║íi Gen3")
	fmt.Println("             40HXInstaller.exe -probe-30hx      # (CMP 30HX) ─Éß╗ìc thanh ghi BAR0 MMIO chß║⌐n ─æo├ín")
	fmt.Println("             40HXInstaller.exe -uninstall       # Gß╗í c├ái ─æß║╖t / Kh├┤i phß╗Ñc hß╗ç thß╗æng")
	fmt.Println("             40HXInstaller.exe -status          # Kiß╗âm tra trß║íng th├íi hiß╗çn tß║íi")
}

// ===================== σ║òσ▒é =====================

func isAdmin() bool {
	return hxcore.IsAdmin()
}

// enableGsp: Φ«╛ EnableGpuFirmware=1 (Θ£Çτ«íτÉåσæÿ)
func enableGsp() error {
	key := hxcore.FindGpuClassKey()
	if key == "" {
		return errors.New("µë╛Σ╕ìσê░ 40HX τÜäΦ«╛σñçµ│¿σåîΦí¿Θö« (Class σ¡ÉΘö«)")
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetDWordValue(gpuEnableFw, 1)
}

// disableGsp: σêá EnableGpuFirmware (σì╕Φ╜╜τö¿, µüóσñìΘ╗ÿΦ«ñσà│)
func disableGsp() {
	key := hxcore.FindGpuClassKey()
	if key == "" {
		return
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	k.DeleteValue(gpuEnableFw)
}

// ensureGspSilent: τí«Σ┐¥ GSP σÉ»τö¿ (EnableGpuFirmware=1)πÇé
// Σ╛¢ -gen2(τÖ╗σ╜òΦç¬σÉ»σè¿)Φ░âτö¿: ΦïÑ GSP Φó½µö╣σ¢₧(Γëá1)σêÖΘçìµû░σÉ»τö¿πÇé
// σåÖ HKLM Θ£Çτ«íτÉåσæÿ: σ╜ôσëìµÿ»τ«íτÉåσæÿτ¢┤µÄÑσåÖ; σÉªσêÖµ│¿σåîΣ╕Çµ¼íµÇº SYSTEM Φ«íσêÆΣ╗╗σèí
// (SYSTEM µ¥âΘÖÉσåÖ HKLM µùáΘ£Ç UAC, µùáτ¬ùσÅú)πÇé
// Φ┐öσ¢₧ true = GSP σ╖▓σÉ»τö¿µêûσ╖▓σ«ëµÄÆΘçìΦ«╛πÇé
func ensureGspSilent() bool {
	if hxcore.GspEnabled() {
		return true // σ╖▓σÉ»τö¿
	}
	fmt.Println("[GSP] EnableGpuFirmware Φó½µö╣σ¢₧, Θçìµû░σÉ»τö¿...")
	if isAdmin() {
		if err := enableGsp(); err != nil {
			fmt.Println("[GSP] ΘçìΦ«╛σñ▒Φ┤Ñ:", err)
			return false
		}
		fmt.Println("[GSP] σ╖▓ΘçìΦ«╛ EnableGpuFirmware=1 (ΘçìσÉ»σÉÄ GSP-RM τöƒµòê)")
		return true
	}
	// Θ¥₧τ«íτÉåσæÿ: τö¿ SYSTEM Φ«íσêÆΣ╗╗σèíΣ╕Çµ¼íµÇºΘçìΦ«╛ (µùá UAC σ╝╣τ¬ù)
	exe, _ := os.Executable()
	abs, _ := filepath.Abs(exe)
	tn := "40HXGspEnsure"
	if out, err := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
		"/tr", fmt.Sprintf("\"%s\" -gspensure -silent", abs),
		"/sc", "once", "/st", "00:00", "/ru", "SYSTEM", "/f"); err != nil {
		fmt.Printf("[GSP] Φ«íσêÆΣ╗╗σèíσê¢σ╗║σñ▒Φ┤Ñ: %s\n", strings.TrimSpace(out))
		return false
	}
	hxcore.RunOut("schtasks.exe", "/run", "/tn", tn)
	hxcore.RunOut("schtasks.exe", "/delete", "/tn", tn, "/f")
	fmt.Println("[GSP] σ╖▓ΘÇÜΦ┐ç SYSTEM Σ╗╗σèíΘçìΦ«╛ EnableGpuFirmware=1")
	return true
}

// gspEnsureMain: -gspensure µ¿íσ╝Å (SYSTEM Φ«íσêÆΣ╗╗σèíΦ░âτö¿, σÅ¬ΘçìΦ«╛ GSP σÉÄΘÇÇσç║)
func gspEnsureMain() {
	if isAdmin() {
		if err := enableGsp(); err != nil {
			fmt.Println("[GSP] gspensure ΘçìΦ«╛σñ▒Φ┤Ñ:", err)
			return
		}
		fmt.Println("[GSP] gspensure: EnableGpuFirmware=1 σ╖▓Φ«╛τ╜«")
	}
}

func resetPnpMain() {
	if !isAdmin() {
		return
	}
	bus := &hxcore.ProductionBus{}
	bus.PnpResetDevice(0x1F0B) // 40HX only; 30HX is protected from PnP resets
}

func copyEmbedTo(target string, src string) error {
	data, err := embedded.ReadFile("embed/" + src)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

// deployEspEfi: σÅîΦ╖»Θâ¿τ╜▓ 40HXUNLK.EFI σê░σ╖▓µîéΦ╜╜τÜä ESP <esp>πÇé
//
//	A. \EFI\40HX\40HXUNLK.EFI   ΓÇö BCD σÉ»σè¿Θí╣σ╝òτö¿Φ╖»σ╛ä
//	B. \EFI\Boot\bootx64.efi    ΓÇö UEFI µáçσçåσ¢₧ΘÇÇΦ╖»σ╛ä (σ¢║Σ╗╢µùáµ¥íΣ╗╢σ░¥Φ»òτÜäµ£ÇσÉÄµëïµ«╡;
//	   Φºúσå│τñ╛σî║σñºΘçÅ"Φúàσ«îΘçìσÉ»τ¢┤µÄÑΦ┐¢ Windows µ▓íΦ╖æΦºúΘöü"ΓÇöΓÇöΣ╕╗µ¥┐σ┐╜τòÑΘ¥₧µáçσçåτ¢«σ╜ò)
//
// σñçΣ╗╜ΦºäσêÖ: ΦïÑτ¢«µáç bootx64.efi σ¡ÿσ£¿Σ╕öΣ╕ìµÿ»µ£¼σ╖Ñσà╖Θâ¿τ╜▓Φ┐çτÜäσë»µ£¼, σàêσñçΣ╗╜Σ╕║
//
//	bootx64.efi.40hx.bak (σì╕Φ╜╜µù╢µüóσñì)πÇéσ╖▓Θâ¿τ╜▓Φ┐ç(.bak σ╖▓σ¡ÿσ£¿)σêÖτ¢┤µÄÑΦªåτ¢ûπÇé
//
// Φ┐öσ¢₧ fallback µÿ»σÉªµû░σñçΣ╗╜Σ║åσÄƒµûçΣ╗╢πÇé
func deployEspEfi(esp string) (backedUp bool, err error) {
	// Φ»╗σÅû embed Σ╕Çµ¼í, Σ╕ñΣ╕¬Φ╖»σ╛äσà▒τö¿
	data, rerr := embedded.ReadFile("embed/40HXUNLK.EFI")
	if rerr != nil {
		return false, rerr
	}
	// σåÖτ¢ÿσëìµáíΘ¬î embed µò░µì«µ£¼Φ║½σ«îµò┤ (PE σñ┤ + Θò┐σ║ªσÉêτÉå, Θÿ▓ embed µìƒσ¥Å)
	if len(data) < 0x2000 { // < 8KB τÜä EFI µûçΣ╗╢σ┐àΣ╕║µìƒσ¥Å
		return false, fmt.Errorf("σåàσ╡î 40HXUNLK.EFI µò░µì«σ╝éσ╕╕ (%d bytes)", len(data))
	}
	if !bytes.HasPrefix(data, []byte("MZ")) {
		return false, errors.New("σåàσ╡î 40HXUNLK.EFI Σ╕ìµÿ»µ£ëµòê PE Θò£σâÅ(τ╝║ MZ σñ┤)")
	}

	// A. Σ╕╗Φ╖»σ╛ä
	dirA := esp + ":" + efiDir // Y:\EFI\40HX
	if merr := os.MkdirAll(dirA, 0o644); merr != nil {
		return false, merr
	}
	pA := filepath.Join(dirA, efiFile)
	if werr := writeVerified(pA, data); werr != nil {
		// σåÖσñ▒Φ┤ÑµêûµáíΘ¬îΣ╕ìΣ╕ÇΦç┤ ΓåÆ σêáµÄëσÅ»Φâ╜σìèµê¬τÜäµûçΣ╗╢, Θü┐σàìΦó½ BCD σ╝òτö¿µêÉσ¥Åσ╝òσ»╝
		os.Remove(pA)
		return false, werr
	}
	fmt.Printf("    [A] %s  (%d bytes, µáíΘ¬î OK)\n", "\\EFI\\40HX\\"+efiFile, len(data))

	// B. µáçσçåσ¢₧ΘÇÇΦ╖»σ╛ä
	dirB := esp + ":" + efiStdDir // Y:\EFI\Boot
	if merr := os.MkdirAll(dirB, 0o644); merr != nil {
		return false, merr
	}
	pB := filepath.Join(dirB, efiStdF) // bootx64.efi
	pBak := pB + efiBakExt             // bootx64.efi.40hx.bak
	if _, berr := os.Stat(pBak); berr != nil {
		// µùáσñçΣ╗╜Φ«░σ╜ò ΓåÆ ΦïÑτ¢«µáçσ¡ÿσ£¿Σ╕öΣ╕ìµÿ»µêæΣ╗¼σ╖▓Θâ¿τ╜▓τÜäσë»µ£¼, σàêσñçΣ╗╜
		if old, oerr := os.ReadFile(pB); oerr == nil && !bytes.Equal(old, data) {
			if cerr := os.Rename(pB, pBak); cerr != nil {
				return false, fmt.Errorf("σñçΣ╗╜σÄƒ %s σñ▒Φ┤Ñ: %v", pB, cerr)
			}
			fmt.Printf("    [B] σÄƒ %s σ╖▓σñçΣ╗╜Σ╕║ %s\n", efiStdF, efiStdF+efiBakExt)
			backedUp = true
		} else if oerr != nil {
			// τ¢«µáçΣ╕ìσ¡ÿσ£¿: µùáσñçΣ╗╜(µ£¼µ¥Ñσ░▒µÿ»τ⌐║Σ╜ì)
		}
	}
	if werr := writeVerified(pB, data); werr != nil {
		os.Remove(pB)
		return backedUp, werr
	}
	fmt.Printf("    [B] %s  (%d bytes, µáíΘ¬î OK)\n", "\\EFI\\Boot\\"+efiStdF, len(data))
	return backedUp, nil
}

// writeVerified: σåÖµûçΣ╗╢σÉÄτ½ïσì│Φ»╗σ¢₧µ»öσ»╣ ΓÇö Θÿ▓µ¡óσåÖσàÑΣ╕¡µû¡/σìèµê¬σ»╝Φç┤σ╝òσ»╝µìƒσ¥ÅπÇé
// Σ╕ìΣ╕ÇΦç┤σêÖσêáΘÖñσ╣╢Φ┐öσ¢₧ΘöÖΦ»»(Φ░âτö¿µû╣µì«µ¡ñΣ╕¡µ¡ó, Σ╕ìΦ«⌐σ¥ÅµûçΣ╗╢τòÖσ£¿σ╝òσ»╝Φ╖»σ╛ä)πÇé
func writeVerified(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	rb, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("σåÖσÉÄµáíΘ¬îΦ»╗σÅûσñ▒Φ┤Ñ %s: %v", path, err)
	}
	if !bytes.Equal(rb, data) {
		return fmt.Errorf("σåÖσÉÄµáíΘ¬îΣ╕ìΣ╕ÇΦç┤ %s (%d Γëá %d bytes)", path, len(rb), len(data))
	}
	return nil
}

// alreadyInstalled: µúÇµ╡ïµÿ»σÉªσ╖▓σ«ëΦúàΦ┐ç(Θü┐σàìµùáµäÅΣ╣ë/ΘçìσñìτÜäΦªåτ¢ûσ«ëΦúà)πÇé
// σêñµì«: Γæá σ¢║Σ╗╢σÉ»σè¿Θí╣ "40HX Unlock" σ¡ÿσ£¿; Γæí ESP Σ╕èσ╖▓µ£ë \EFI\40HX\40HXUNLK.EFIπÇé
// Σ╗╗Σ╕Çσæ╜Σ╕¡σì│Φ«ñΣ╕║ΦúàΦ┐ç ΓÇö τö¿Σ║ÄΘçìσàÑµÅÉτñ║(Σ╕ìΣ╝Üσ¢áµ¡ñΘÿ╗µ¡óτö¿µê╖, Σ╗àσ╝╣τí«Φ«ñ)πÇé
func alreadyInstalled() bool {
	// Γæá bcdedit σ¢║Σ╗╢µ₧ÜΣ╕╛(Σ╕ìµîé ESP, σ┐½ΘÇƒ)
	if out, _ := hxcore.RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc) {
		return true
	}
	// Γæí ESP µûçΣ╗╢
	esp := hxcore.MountESP()
	if esp == "" {
		return false // µîéΣ╕ìΣ╕è ESP µù╢Σ┐¥σ«êΦºåΣ╕║µ£¬Φúà(σÉÄΘ¥ó [5/8] Σ╝ÜµèÑΘöÖσ╝òσ»╝)
	}
	defer hxcore.UnmountESP(esp)
	if _, err := os.Stat(esp + ":" + efiDir + "\\" + efiFile); err == nil {
		return true
	}
	return false
}

// verifyBootEntry: Φ»╗σ¢₧ {fwbootmgr} displayorder, τí«Φ«ñ 40HX Unlock µÿ»σÉªσ£¿ΘªûΣ╜ìπÇé
// Φ┐öσ¢₧ (exists, isFirst, displayOrderµÅÅΦ┐░)πÇé
// τö¿ bcdedit /enum firmware Φ»╗σ¢║Σ╗╢ NVRAM ΓÇö ΦïÑσ¢║Σ╗╢σ┐╜τòÑ bcdedit τÜäσåÖσàÑ,
// Φ┐ÖΘçîΣ╝Üσªéσ«₧σÅìµÿá(Σ╕ìσ£¿σêùΦí¿/Σ╕ìσ£¿ΘªûΣ╜ì), Σ╗ÄΦÇîΦ«⌐σ«ëΦúàσÖ¿τ╗Öσç║ BIOS µëïσè¿µîçσ╝òπÇé
// µ│¿µäÅ: bcdedit Φ╛ôσç║Σ╕║ GBK, Σ╕¡µûçτ│╗τ╗ƒ"µáçΦ»åτ¼ª/Φ»┤µÿÄ"µÿ»Σ╣▒τáü; Σ╜åσ¡ùµ«╡σÇ╝
// (guid / displayorder / 40HX Unlock / path) σ¥çΣ╕║ ASCII, µîëσ¥ùΦºúµ₧ÉσÅ»Θ¥áπÇé
func verifyBootEntry() (bool, bool, string) {
	out, err := hxcore.RunOut("bcdedit.exe", "/enum", "firmware")
	if err != nil {
		return false, false, "(bcdedit Φ»╗σÅûσñ▒Φ┤Ñ: " + err.Error() + ")"
	}
	lines := strings.Split(out, "\r\n")
	if len(lines) < 2 {
		lines = strings.Split(out, "\n")
	}

	// 1. µö╢Θ¢å displayorder Σ╕ïτÜä GUID σ║Åσêù(σ¢║Σ╗╢σ«₧ΘÖàσÉ»σè¿Θí║σ║Å)
	var order []string
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "displayorder") {
			// ΘªûΣ╕¬ GUID σÅ»Φâ╜σÉîΦíî: "displayorder {guid}"
			if m := guidRe().FindString(t); m != "" {
				order = append(order, strings.Trim(m, "{}"))
			}
			// σÉÄτ╗¡τ╝⌐Φ┐¢Φíî {guid}
			for j := i + 1; j < len(lines); j++ {
				s := strings.TrimSpace(lines[j])
				if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
					order = append(order, strings.Trim(s, "{}"))
				} else if s != "" {
					break
				}
			}
			break // displayorder σÅ¬σ£¿ {fwbootmgr} µ«╡, σÅûΘªûΣ╕¬σì│σÅ»
		}
	}

	// 2. µë╛ description Σ╕║ "40HX Unlock" τÜäσ¥ùτÜä GUID
	target := ""
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "description") &&
			strings.Contains(lines[i], bootDesc) {
			// σ╛ÇΣ╕èµë╛µ£ÇΦ┐æτÜä {guid} Φíî = Φ»Ñσ¥ù identifier
			for j := i - 1; j >= 0 && j > i-6; j-- {
				if m := guidRe().FindString(lines[j]); m != "" {
					target = strings.Trim(m, "{}")
					break
				}
			}
			break
		}
	}
	if target == "" {
		joined := strings.Join(order, " > ")
		if joined == "" {
			joined = "(σ¢║Σ╗╢µùá displayorder µ¥íτ¢«)"
		}
		return false, false, joined
	}
	if len(order) == 0 {
		return true, false, "(displayorder Σ╕║τ⌐║)"
	}
	isFirst := order[0] == target
	return true, isFirst, strings.Join(order, " > ")
}

var _guidRe = regexp.MustCompile(`\{([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\}`)

func guidRe() *regexp.Regexp { return _guidRe }

// ===================== σ«ëΦúà =====================

// applyPowerSettings: σ┐½ΘÇƒσÉ»σè¿ + PCIe ASPM Σ╕ñΘí╣τö╡µ║ÉΣ╝ÿσîû(v2.6.0 [3.6/8] µ«╡µè╜σÅû,
// v2.6.0 GUI τ¡ûτòÑΘí╡σñìτö¿)πÇéσ╣éτ¡ë: σÄƒµ£¼σ╖▓σà│σêÖΣ╕ìσè¿; Φ┐öσ¢₧ΘÇÉΘí╣Φ»┤µÿÄΦíîπÇé
func applyPowerSettings() []string {
	notes := []string{}
	if hxcore.FastStartupOn() {
		if err := hxcore.SetFastStartupOff(); err != nil {
			notes = append(notes, fmt.Sprintf("σ┐½ΘÇƒσÉ»σè¿σà│Θù¡σñ▒Φ┤Ñ: %v (Σ╕ìσ╜▒σôìσ«ëΦúà, σ╗║Φ««τö╡µ║ÉΘÇëΘí╣µëïσè¿σà│)", err))
		} else {
			notes = append(notes, "σ┐½ΘÇƒσÉ»σè¿σ╖▓σà│Θù¡(σÄƒΣ╕║σ╝Ç): σà│µ£║σ░åΦ╡░σ«îµò┤ UEFI σ╝òσ»╝; τö╡µ║ÉΘÇëΘí╣σÅ»µüóσñì")
		}
	} else {
		notes = append(notes, "σ┐½ΘÇƒσÉ»σè¿: σÄƒµ£¼σ╖▓σà│(OK)")
	}
	if ac, dc, ok := hxcore.ASPMSavings(); !ok {
		notes = append(notes, "PCIe ASPM: µ£¼µ£║µ£¬σà¼σ╝ÇΦ»ÑΦ«╛τ╜«, Φ╖│Φ┐ç")
	} else if ac == 0 && dc == 0 {
		notes = append(notes, "PCIe ASPM: σÄƒµ£¼σ╖▓σà│(OK)")
	} else {
		if err := hxcore.SetASPMOff(); err != nil {
			notes = append(notes, fmt.Sprintf("ASPM σà│Θù¡σñ▒Φ┤Ñ: %v", err))
		} else {
			notes = append(notes, fmt.Sprintf("PCIe ASPM σ╖▓σà│Θù¡(σÄƒ AC=%d/DC=%d): σçÅσ░æτ⌐║Θù▓ΘÖìσê░ Gen1; µüóσñì: powercfg σæ╜Σ╗ñΦºü README", ac, dc))
		}
	}
	return notes
}

// installEFI: ESP σÅîΦ╖»Θâ¿τ╜▓ 40HXUNLK.EFI + σ¢║Σ╗╢σÉ»σè¿Θí╣(v2.6.0 [5/8]+[6/8] µ«╡µè╜σÅû,
// v2.6.0 GUI τ╗äΣ╗╢σ«ëΦúàΘí╡σñìτö¿)πÇéΦ┐öσ¢₧ EFI µÿ»σÉªΘâ¿τ╜▓µêÉσèƒ;
// [7/8] Gen2 Σ╗╗σèíµ│¿σåîΣ╕ìΣ╛¥Φ╡ûµ¡ñτ╗ôµ₧£(EFI σñ▒Φ┤ÑσÅ¬Φ╖│Φ┐ç EFI Σ╕ñµ¡Ñ ΓÇö τñ╛σî║ #2/#5/#6/#7 τ╗ƒΣ╕Çµá╣σ¢áΣ┐«σñì)πÇé
func installEFI() bool {
	//    Σ╕╗Φ╖»σ╛ä  \EFI\40HX\40HXUNLK.EFI  ΓÇö BCD σÉ»σè¿Θí╣σ╝òτö¿
	//    fallback \EFI\Boot\bootx64.efi   ΓÇö UEFI µáçσçåσ¢₧ΘÇÇΦ╖»σ╛ä, Φºúσå│Θâ¿σêåΣ╕╗µ¥┐
	//    σ┐╜τòÑ BCD displayorder / Σ╕ìΦ«ñΘ¥₧µáçσçåτ¢«σ╜ò(τñ╛σî║"Φúàσ«îΘçìσÉ»µ▓íσÅìσ║ö"Σ╕╗σ¢á)πÇé
	//    σÄƒ bootx64.efi σñçΣ╗╜Σ╕║ bootx64.efi.40hx.bak, σì╕Φ╜╜µù╢µüóσñìπÇé
	efiOK := false
	fmt.Println("    ┬╖ Triß╗ân khai EFI mß╗ƒ kho├í v├áo ph├ón v├╣ng EFI hß╗ç thß╗æng (2 ─æ╞░ß╗¥ng dß║½n)...")
	esp := hxcore.MountESP()
	if esp == "" {
		if hxcore.FirmwareIsLegacy() {
			fmt.Println("[!] Hß╗ç thß╗æng ─æang khß╗ƒi ─æß╗Öng kiß╗âu BIOS c┼⌐ (Legacy)+MBR ΓÇö Kh├┤ng c├│ ph├ón v├╣ng EFI, kh├┤ng thß╗â triß╗ân khai EFI mß╗ƒ kho├í.")
			fmt.Println("    Mß╗ƒ kho├í hiß╗çu n─âng cß║ºn UEFI+GPT: Vui l├▓ng chuyß╗ân ─æß╗òi ß╗ò ─æ─⌐a sang GPT bß║▒ng c├┤ng cß╗Ñ mbr2gpt (xem h╞░ß╗¢ng dß║½n trong hß╗Öp thoß║íi),")
			fmt.Println("    Sau khi chuyß╗ân ─æß╗òi v├á bß║¡t UEFI trong BIOS, h├úy chß║íy lß║íi tr├¼nh c├ái ─æß║╖t n├áy.")
			fmt.Println("    [i] T├íc vß╗Ñ tß╗▒ mß╗ƒ kho├í Gen2 khi ─æ─âng nhß║¡p kh├┤ng bß╗ï ß║únh h╞░ß╗ƒng, tiß║┐p tß╗Ñc ─æ─âng k├╜ (xem [7/8]).")
			msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (Cß║ºn chuyß╗ân ─æß╗òi ß╗ò ─æ─⌐a sang GPT)",
				"Hß╗ç thß╗æng cß╗ºa bß║ín ─æang chß║íy ß╗ƒ chß║┐ ─æß╗Ö BIOS c┼⌐ (Legacy)+MBR, kh├┤ng c├│ ph├ón v├╣ng EFI,\n"+
					"n├¬n kh├┤ng thß╗â c├ái ─æß║╖t EFI mß╗ƒ kho├í hiß╗çu n─âng.\n\n"+
					"Vui l├▓ng chuyß╗ân ─æß╗òi sang UEFI+GPT (c├┤ng cß╗Ñ mbr2gpt ch├¡nh thß╗⌐c cß╗ºa Microsoft, kh├┤ng mß║Ñt dß╗» liß╗çu):\n"+
					"  1. Sao l╞░u dß╗» liß╗çu quan trß╗ìng; ─Éß║úm bß║úo BitLocker ─æ├ú tß║»t hoß║╖c tß║ím ng╞░ng\n"+
					"  2. Mß╗ƒ CMD vß╗¢i quyß╗ün Admin v├á chß║íy:  mbr2gpt /validate /allowfullos\n"+
					"  3. Khi hiß╗çn 'Validation completed successfully', chß║íy tiß║┐p:\n"+
					"        mbr2gpt /convert /allowfullos\n"+
					"  4. Khß╗ƒi ─æß╗Öng lß║íi v├áo BIOS, chuyß╗ân chß║┐ ─æß╗Ö Boot tß╗½ Legacy sang UEFI (Tß║»t CSM)\n"+
					"  5. V├áo Windows v├á chß║íy lß║íi tr├¼nh c├ái ─æß║╖t n├áy\n\n"+
					"L╞░u ├╜: Qu├í tr├¼nh chuyß╗ân ─æß╗òi kh├┤ng thß╗â ─æß║úo ng╞░ß╗úc; y├¬u cß║ºu Windows 10 1703+ / Win 11 v├á bo mß║ích chß╗º hß╗ù trß╗ú UEFI.\n"+
					"Tr├¼nh c├ái ─æß║╖t sß║╜ tiß║┐p tß╗Ñc ─æ─âng k├╜ phß║ºn tß╗▒ mß╗ƒ kho├í Gen2.",
				mbIconWarn)
		} else {
			fmt.Println("[!] Kh├┤ng thß╗â gß║»n kß║┐t ph├ón v├╣ng EFI (mountvol /S thß║Ñt bß║íi)")
			fmt.Println("    Hß╗ç thß╗æng l├á UEFI, nguy├¬n nh├ón phß╗ò biß║┐n: BitLocker ─æang bß║¡t hoß║╖c ph├ón v├╣ng ESP c├│ bß║Ñt th╞░ß╗¥ng.")
			fmt.Println("    C├│ thß╗â c├ái thß╗º c├┤ng: mountvol S: /S, ch├⌐p 40HXUNLK.EFI v├áo S:\\EFI\\40HX\\, mountvol S: /D")
			msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (Gß║»n kß║┐t ph├ón v├╣ng EFI thß║Ñt bß║íi)",
				"Kh├┤ng thß╗â gß║»n kß║┐t ph├ón v├╣ng EFI (mountvol /S thß║Ñt bß║íi), EFI mß╗ƒ kho├í ch╞░a ─æ╞░ß╗úc triß╗ân khai lß║ºn n├áy.\n"+
					"Khß╗ƒi ─æß╗Öng hß╗ç thß╗æng kh├┤ng bß╗ï ß║únh h╞░ß╗ƒng.\n\n"+
					"Nguy├¬n nh├ón phß╗ò biß║┐n: BitLocker/m├ú h├│a b├¬n thß╗⌐ ba ─æang bß║¡t, ph├ón v├╣ng ESP bß║Ñt th╞░ß╗¥ng.\n"+
					"Bß║ín c├│ thß╗â sao ch├⌐p thß╗º c├┤ng (xem nhß║¡t k├╜ v├á t├ái liß╗çu H╞░ß╗¢ng Dß║½n Sß╗¡a Lß╗ùi EFI).\n\n"+
					"Tr├¼nh c├ái ─æß║╖t sß║╜ tiß║┐p tß╗Ñc ─æ─âng k├╜ phß║ºn tß╗▒ mß╗ƒ kho├í Gen2.",
				mbIconWarn)
		}
		return false
	}
	fmt.Printf("    ESP gß║»n kß║┐t tß║íi %s: \\\n", esp)
	fb, err := deployEspEfi(esp)
	hxcore.UnmountESP(esp)
	if err != nil {
		fmt.Println("[!] Sao ch├⌐p EFI thß║Ñt bß║íi:", err)
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (Ghi EFI thß║Ñt bß║íi)",
			"Sao ch├⌐p EFI mß╗ƒ kho├í v├áo ESP thß║Ñt bß║íi:\n"+err.Error()+
				"\n\nKhß╗ƒi ─æß╗Öng hß╗ç thß╗æng kh├┤ng bß╗ï ß║únh h╞░ß╗ƒng, m├íy vß║½n c├│ thß╗â khß╗ƒi ─æß╗Öng v├áo Windows b├¼nh th╞░ß╗¥ng.\n\n"+
				"Nß║┐u muß╗æn triß╗ân khai thß╗º c├┤ng, xem mß╗Ñc C├ái ─æß║╖t thß╗º c├┤ng trong H╞░ß╗¢ng Dß║½n Cß╗⌐u Hß╗Ö EFI.\n\n"+
				"Tr├¼nh c├ái ─æß║╖t sß║╜ tiß║┐p tß╗Ñc ho├án th├ánh phß║ºn Gen2.", mbIconWarn)
		return false
	}
	if fb {
		fmt.Println("    [!] Ph├ít hiß╗çn file bootx64.efi gß╗æc, ─æ├ú sao l╞░u th├ánh bootx64.efi.40hx.bak")
	}
	efiOK = true

	// BootOrder (v2.4: σåÖσ¢₧Θ¬îΦ»ü + BIOS µîçσ╝òσ╝╣µíå); Σ╗à EFI Θâ¿τ╜▓µêÉσèƒµëìµëºΦíî
	// BootOrder: Thiß║┐t lß║¡p mß╗Ñc khß╗ƒi ─æß╗Öng ╞░u ti├¬n sß╗æ 1
	fmt.Println("    ┬╖ Thiß║┐t lß║¡p mß╗Ñc khß╗ƒi ─æß╗Öng firmware ('40HX Unlock' ╞░u ti├¬n h├áng ─æß║ºu)...")
	bootOK := false
	if err := setupBootEntry(); err != nil {
		fmt.Println("[!] Tß╗▒ ─æß╗Öng thiß║┐t lß║¡p mß╗Ñc khß╗ƒi ─æß╗Öng thß║Ñt bß║íi:", err)
	} else {
		if ex, first, ord := verifyBootEntry(); ex {
			bootOK = first
			if first {
				fmt.Println("    Mß╗Ñc khß╗ƒi ─æß╗Öng ─æ├ú ─æ╞░ß╗úc ─æß║╖t ─æß║ºu ti├¬n v├á x├íc minh th├ánh c├┤ng (displayorder h├áng ─æß║ºu)")
			} else {
				fmt.Println("    [!] Mß╗Ñc khß╗ƒi ─æß╗Öng ─æ├ú ─æ╞░ß╗úc tß║ío, nh╞░ng ch╞░a nß║▒m ─æß║ºu ti├¬n trong displayorder:")
				fmt.Println("        Thß╗⌐ tß╗▒ khß╗ƒi ─æß╗Öng hiß╗çn tß║íi: " + ord)
				fmt.Println("        Vui l├▓ng v├áo BIOS ─æß║╖t '40HX Unlock' l├ám mß╗Ñc khß╗ƒi ─æß╗Öng ─æß║ºu ti├¬n (xem h╞░ß╗¢ng dß║½n)")
			}
		} else {
			fmt.Println("    [!] Kh├┤ng t├¼m thß║Ñy mß╗Ñc '40HX Unlock' trong danh s├ích khß╗ƒi ─æß╗Öng firmware")
			fmt.Println("        (Mß╗Öt sß╗æ bo mß║ích chß╗º bß╗Å qua lß╗çnh ghi BCD, vui l├▓ng v├áo BIOS ─æß╗â th├¬m/─æß║╖t l├¬n ─æß║ºu)")
		}
	}
	if !bootOK {
		// H╞░ß╗¢ng dß║½n BIOS
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (L╞░u ├╜ quan trß╗ìng: Vui l├▓ng l├ám theo h╞░ß╗¢ng dß║½n)",
			"Mß╗Ñc khß╗ƒi ─æß╗Öng tß╗▒ ─æß╗Öng ch╞░a ─æ╞░ß╗úc firmware bo mß║ích chß╗º chß║Ñp nhß║¡n ho├án to├án.\n"+
				"Vui l├▓ng khß╗ƒi ─æß╗Öng lß║íi m├íy, bß║Ñm Del/F2 v├áo BIOS v├á thß╗▒c hiß╗çn c├íc thiß║┐t lß║¡p sau:\n\n"+
				"1. Tß║»t Secure Boot (nß║┐u bß║¡t, file EFI ch╞░a k├╜ sß╗æ sß║╜ bß╗ï chß║╖n)\n"+
				"2. Tß║»t Fast Boot trong BIOS (nß║┐u c├│)\n"+
				"3. Trong mß╗Ñc [Thß╗⌐ tß╗▒ khß╗ƒi ─æß╗Öng / Boot Priority], ─æß║╖t '40HX Unlock' l├¬n vß╗ï tr├¡ ─æß║ºu ti├¬n\n"+
				"   hoß║╖c chß╗ìn khß╗ƒi ─æß╗Öng thß╗º c├┤ng tß╗½ file \\EFI\\40HX\\40HXUNLK.EFI\n"+
				"4. Nß║┐u danh s├ích chß╗ë c├│ Windows Boot Manager:\n"+
				"   - Mß╗Öt sß╗æ bo mß║ích chß╗º cß║ºn tß║»t CSM (chuyß╗ân sang thuß║ºn UEFI) mß╗¢i hiß╗çn mß╗Ñc n├áy\n"+
				"   - Hoß║╖c chß╗ìn khß╗ƒi ─æß╗Öng trß╗▒c tiß║┐p tß╗½ ph├ón v├╣ng UEFI (chß║íy qua bß║ún dß╗▒ ph├▓ng bootx64)\n\n"+
				"Tr├¼nh c├ái ─æß║╖t ─æ├ú triß╗ân khai payload mß╗ƒ kho├í v├áo cß║ú 2 vß╗ï tr├¡:\n"+
				"  \\EFI\\40HX\\40HXUNLK.EFI  (─É╞░ß╗¥ng dß║½n BCD ch├¡nh)\n"+
				"  \\EFI\\Boot\\bootx64.efi    (─É╞░ß╗¥ng dß║½n dß╗▒ ph├▓ng chuß║⌐n)\n\n"+
				"Nhß║¡t k├╜ chi tiß║┐t: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
			mbIconError)
	}
	return efiOK
}

func install() {
	fmt.Println("==============================================")
	fmt.Println("  Tr├¼nh C├ái ─Éß║╖t Mß╗ƒ Kho├í CMP 40HX / 30HX Windows v3.0.0")
	fmt.Println("  Mß╗ƒ kho├í Tensor (EFI V70 + Bß║¡t GSP) + PCIe Gen2 + Tß╗▒ khß╗ƒi ─æß╗Öng")
	fmt.Println("==============================================")

	if !isAdmin() {
		fmt.Println("[!] Cß║ºn quyß╗ün Quß║ún trß╗ï vi├¬n (Administrator).")
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "Cß║ºn quyß╗ün Quß║ún trß╗ï vi├¬n.\nVui l├▓ng nhß║Ñp chuß╗Öt phß║úi -> Chß╗ìn Run as administrator.", mbIconError)
		return
	}
	if lockOnce(`Local\40HXInstaller_v1`) == nil {
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "Tr├¼nh c├ái ─æß║╖t ─æang chß║íy, vui l├▓ng kh├┤ng nhß║Ñp tr├╣ng lß║╖p.", mbIconInfo)
		return
	}

	// 0. Kiß╗âm tra c├ái ─æß║╖t tr╞░ß╗¢c ─æ├│
	if alreadyInstalled() {
		fmt.Println("[!] Ph├ít hiß╗çn mß╗ƒ kho├í 40HX ─æ├ú ─æ╞░ß╗úc c├ái ─æß║╖t tr╞░ß╗¢c ─æ├│ (mß╗Ñc khß╗ƒi ─æß╗Öng/kho├í GSP ─æ├ú tß╗ôn tß║íi).")
		if !msgboxYesNo("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX",
			"Ph├ít hiß╗çn mß╗ƒ kho├í 40HX ─æ├ú ─æ╞░ß╗úc c├ái ─æß║╖t tr├¬n hß╗ç thß╗æng n├áy.\n\n"+
				"C├ái ─æß║╖t lß║íi sß║╜ ghi ─æ├¿ thiß║┐t lß║¡p hiß╗çn c├│ (driver v├á mß╗Ñc khß╗ƒi ─æß╗Öng sß║╜ ─æ╞░ß╗úc cß║¡p nhß║¡t, kh├┤ng ß║únh h╞░ß╗ƒng khß╗ƒi ─æß╗Öng Windows).\n"+
				"Nß║┐u bß║ín muß╗æn sß╗¡a lß╗ùi hoß║╖c n├óng cß║Ñp, chß╗ìn \"Yes\" ─æß╗â tiß║┐p tß╗Ñc;\n"+
				"Nß║┐u chß╗ë v├┤ t├¼nh mß╗ƒ, chß╗ìn \"No\" ─æß╗â giß╗» nguy├¬n trß║íng th├íi.\n\n"+
				"Bß║ín c├│ muß╗æn tiß║┐p tß╗Ñc c├ái ─æß║╖t lß║íi?") {
			fmt.Println("─É├ú huß╗╖ ΓÇö Giß╗» nguy├¬n trß║íng th├íi c├ái ─æß║╖t hiß╗çn tß║íi.")
			return
		}
		fmt.Println("    Ng╞░ß╗¥i d├╣ng x├íc nhß║¡n, tiß║┐p tß╗Ñc c├ái ─æß║╖t ghi ─æ├¿.")
	}

	// 1. Kiß╗âm tra GPU
	fmt.Print("[1/8] Kiß╗âm tra GPU ... ")
	if !hxcore.FindGPU() {
		fmt.Println("Kh├┤ng t├¼m thß║Ñy " + gpuVenDev)
		fmt.Println("[!] Kh├┤ng ph├ít hiß╗çn card CMP 40HX / 30HX. Dß╗½ng qu├í tr├¼nh.")
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "Kh├┤ng t├¼m thß║Ñy card m├án h├¼nh CMP 40HX / 30HX t╞░╞íng th├¡ch.\nQu├í tr├¼nh c├ái ─æß║╖t ─æ├ú dß╗½ng lß║íi.", mbIconError)
		return
	}
	fmt.Println("─É├ú t├¼m thß║Ñy GPU t╞░╞íng th├¡ch!")

	// 2. Secure Boot
	fmt.Print("[2/8] Kiß╗âm tra Secure Boot ... ")
	if hxcore.SecureBootOn() {
		fmt.Println("─Éang Bß║¼T!")
		fmt.Println("[!] Secure Boot ─æang bß║¡t, file EFI mß╗ƒ kho├í ch╞░a k├╜ sß║╜ bß╗ï BIOS tß╗½ chß╗æi nß║íp.")
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (Cß║ºn tß║»t Secure Boot)",
			"Ph├ít hiß╗çn Secure Boot ─æang Bß║¼T, file EFI mß╗ƒ kho├í sß║╜ bß╗ï BIOS tß╗½ chß╗æi nß║íp.\n\n"+
				"Vui l├▓ng v├áo BIOS tß║»t Secure Boot tr╞░ß╗¢c khi chß║íy bß╗Ö c├ái:\n"+
				"  1. Khß╗ƒi ─æß╗Öng lß║íi m├íy, bß║Ñm Del / F2 (hoß║╖c F1/F10/F12 tuß╗│ bo mß║ích chß╗º)\n"+
				"  2. T├¼m mß╗Ñc Security / Boot\n"+
				"  3. Chuyß╗ân Secure Boot sang Disabled\n"+
				"  4. Bß║Ñm F10 l╞░u v├á khß╗ƒi ─æß╗Öng lß║íi v├áo Windows\n\n"+
				"─É├óy l├á b╞░ß╗¢c bß║»t buß╗Öc v├¼ EFI mß╗ƒ kho├í kh├┤ng c├│ chß╗» k├╜ sß╗æ cß╗ºa Microsoft.",
			mbIconError)
		return
	}
	fmt.Println("─É├ú tß║»t / Kh├┤ng khß║ú dß╗Ñng (OK)")

	// 3. Test Signing
	fmt.Print("[3/8] Kiß╗âm tra Test Signing ... ")
	if hxcore.TestSigningOn() {
		fmt.Println("─Éang bß║¡t ΓÇö v2.5+ kh├┤ng cß║ºn, c├│ thß╗â tß║»t bß║▒ng: bcdedit /set testsigning off")
	} else {
		fmt.Println("─É├ú tß║»t (OK) ΓÇö v2.5+ ho├án to├án kh├┤ng cß║ºn Test Signing")
	}

	// 3.5 GSP
	fmt.Print("[3.5/8] Bß║¡t GSP (EnableGpuFirmware) ... ")
	if hxcore.GspEnabled() {
		if sub, _, fw := hxcore.GspDiag(); sub != "" {
			fmt.Printf("─É├ú bß║¡t (OK) ΓÇö Class\\%s EnableGpuFirmware=%d\n", sub, fw)
		} else {
			fmt.Println("─É├ú bß║¡t (OK)")
		}
	} else {
		if err := enableGsp(); err != nil {
			_, adapterDiag, _ := hxcore.GspDiag()
			fmt.Println("C├ái ─æß║╖t thß║Ñt bß║íi:", err)
			if adapterDiag != "" && !strings.Contains(adapterDiag, "Kh├┤ng khß╗¢p") {
				fmt.Println("    [!] AdapterString thß╗▒c tß║┐:", adapterDiag)
			}
			msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "Thiß║┐t lß║¡p EnableGpuFirmware=1 thß║Ñt bß║íi (cß║ºn quyß╗ün Admin).\nSau khi mß╗ƒ kho├í c├│ thß╗â bß╗ï lß╗ùi Code 43.\nLß╗ùi: "+err.Error(), mbIconError)
			return
		}
		fmt.Println("─É├ú ─æß║╖t EnableGpuFirmware=1 (C├│ hiß╗çu lß╗▒c sau khi khß╗ƒi ─æß╗Öng lß║íi)")
		fmt.Println("    [!] GSP bß║»t buß╗Öc: Nß║┐u kh├┤ng driver sß║╜ kh├┤ng nhß║¡n trß║íng th├íi mß╗ƒ kho├í -> Lß╗ùi Code 43")
	}

	// 3.6 Nguß╗ôn
	fmt.Print("[3.6/8] Thiß║┐t lß║¡p nguß╗ôn (Khß╗ƒi ─æß╗Öng nhanh + Tiß║┐t kiß╗çm ─æiß╗çn PCIe) ... ")
	pwrNotes := applyPowerSettings()
	fmt.Println("Ho├án th├ánh")
	for _, n := range pwrNotes {
		fmt.Println("    - " + n)
	}

	// 4. Driver
	fmt.Println("[4/8] Chuß║⌐n bß╗ï driver Gen2 BYOVD (ThrottleStop + WinRing0)...")
	installDrivers()

	// 4.5 Defender
	fmt.Print("[4.5/8] Th├¬m loß║íi trß╗½ Defender (chß╗æng xo├í nhß║ºm driver) ... ")
	if err := hxcore.AddDefenderExclusions(); err != nil {
		fmt.Println("Ch╞░a thß╗▒c hiß╗çn (c├│ thß╗â bß╗Å qua):", err)
	} else {
		fmt.Println("─É├ú th├¬m loß║íi trß╗½ cho file driver ThrottleStop/WinRing0")
	}

	// 5+6. EFI & Boot Order
	fmt.Println("[5/8]+[6/8] Triß╗ân khai EFI mß╗ƒ kho├í v├á mß╗Ñc khß╗ƒi ─æß╗Öng firmware...")
	efiOK := installEFI()

	// 7. Gen2 Startup
	fmt.Println("[7/8] ─É─âng k├╜ tß╗▒ mß╗ƒ kho├í Gen2 khi ─æ─âng nhß║¡p (T├íc vß╗Ñ SYSTEM + Kho├í Run)...")
	setRunKey()
	if err := setupGen2Task(); err != nil {
		fmt.Println("[!]", err)
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (─É─âng k├╜ tß╗▒ chß║íy thß║Ñt bß║íi)",
			"─É─âng k├╜ t├íc vß╗Ñ tß╗▒ mß╗ƒ kho├í Gen2 khi ─æ─âng nhß║¡p thß║Ñt bß║íi.\n\n"+
				"Vui l├▓ng chß║íy lß║íi vß╗¢i quyß╗ün Admin bß║▒ng lß╗çnh:\n"+
				"  40HXInstaller.exe -task\n\n"+
				"C├íc b╞░ß╗¢c c├ái ─æß║╖t kh├íc ─æ├ú ho├án th├ánh.", mbIconWarn)
	}

	fmt.Println()
	fmt.Println("C├ái ─æß║╖t ho├án tß║Ñt!")
	if efiOK {
		fmt.Println("  Lß║ºn khß╗ƒi ─æß╗Öng tiß║┐p theo: Firmware sß║╜ tß╗▒ chß║íy '40HX Unlock' (Mß╗ƒ kho├í Tensor) -> V├áo Windows")
	} else {
		fmt.Println("  [!] Ch╞░a triß╗ân khai EFI mß╗ƒ kho├í hiß╗çu n─âng ΓÇö Hiß╗çu n─âng tß║ím thß╗¥i ch╞░a mß╗ƒ kho├í,")
		fmt.Println("      L├ám theo h╞░ß╗¢ng dß║½n (mbr2gpt/c├ái thß╗º c├┤ng) rß╗ôi chß║íy lß║íi bß╗Ö c├ái.")
	}
	fmt.Println("  GSP ─æ├ú bß║¡t: Driver hoß║ít ─æß╗Öng ß╗ƒ chß║┐ ─æß╗Ö GSP-RM, sau mß╗ƒ kho├í kh├┤ng lo lß╗ùi Code 43")
	fmt.Println("  Sau khi ─æ─âng nhß║¡p: Gen2 tß╗▒ ─æß╗Öng mß╗ƒ kho├í (chß║íy ß║⌐n, kh├┤ng hiß╗çn cß╗¡a sß╗ò)")
	fmt.Println("  [!] Qu├í tr├¼nh c├ái ─æß║╖t kh├┤ng ├⌐p xung PCIe ngay, chß╗ë k├¡ch hoß║ít khi ─æ─âng nhß║¡p sau reboot (tr├ính xung ─æß╗Öt driver)")
	fmt.Println("  Kiß╗âm tra sau reboot: Chß║íy 40HXCheck.exe kiß╗âm tra trß║íng th├íi mß╗ƒ kho├í (SS0=0x88888888 l├á th├ánh c├┤ng)")

	efiNote := ""
	if efiOK {
		efiNote = "L╞░u ├╜ khi khß╗ƒi ─æß╗Öng lß║íi m├íy:\n" +
			"  ┬╖ Nß║┐u m├án h├¼nh ─æen / hiß╗ân thß╗ï log chß╗» 40HX khoß║úng 10~30 gi├óy l├á b├¼nh th╞░ß╗¥ng (─æang mß╗ƒ kho├í)\n" +
			"  ┬╖ Sau khi mß╗ƒ kho├í xong sß║╜ tß╗▒ ─æß╗Öng v├áo Windows b├¼nh th╞░ß╗¥ng\n\n" +
			"Nß║┐u khß╗ƒi ─æß╗Öng lß║íi v├áo thß║│ng Windows m├á kh├┤ng qua m├án h├¼nh mß╗ƒ kho├í, h├úy v├áo BIOS (Del/F2):\n" +
			"  1. Tß║»t Secure Boot\n" +
			"  2. Tß║»t Fast Boot\n" +
			"  3. ─Éß║╖t '40HX Unlock' l├ám mß╗Ñc khß╗ƒi ─æß╗Öng ─æß║ºu ti├¬n\n" +
			"     (Nß║┐u danh s├ích chß╗ë c├│ Windows Boot Manager, h├úy tß║»t CSM)\n"
	} else {
		efiNote = "[!] Lß║ºn n├áy ch╞░a triß╗ân khai EFI mß╗ƒ kho├í hiß╗çu n─âng:\n" +
			"  ┬╖ Hiß╗çu n─âng tß║ím thß╗¥i giß╗» nguy├¬n\n" +
			"  ┬╖ T├íc vß╗Ñ tß╗▒ mß╗ƒ kho├í Gen2 ─æ├ú ─æ─âng k├╜ th├ánh c├┤ng, kh├┤ng bß╗ï ß║únh h╞░ß╗ƒng\n"
	}
	msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX (C├ái ─æß║╖t ho├án tß║Ñt)",
		"Γ£à C├ái ─æß║╖t ho├án tß║Ñt! "+map[bool]string{true: "Sau khi khß╗ƒi ─æß╗Öng lß║íi sß║╜ tß╗▒ ─æß╗Öng mß╗ƒ kho├í.", false: "Phß║ºn Gen2 ─æ├ú sß║╡n s├áng."}[efiOK]+"\n\n"+
			efiNote+
			"\nSau khi v├áo lß║íi Windows:\n"+
			"  ┬╖ Chß║íy file 40HXCheck.exe ─æß╗â kiß╗âm tra ΓÇö Nß║┐u b├ío\n"+
			"    'Mß╗ƒ kho├í th├ánh c├┤ng: Tensor tß╗æi ─æa (SS0=0x88888888)' l├á ho├án tß║Ñt\n"+
			"  ┬╖ Nß║┐u b├ío ch╞░a mß╗ƒ kho├í, c├┤ng cß╗Ñ sß║╜ h╞░ß╗¢ng dß║½n b╞░ß╗¢c xß╗¡ l├╜ (bß║¡t Above 4G, v.v.)\n\n"+
			"┬╖ GSP ─æ├ú bß║¡t (EnableGpuFirmware=1)\n"+
			"┬╖ ─É├ú tß║»t Khß╗ƒi ─æß╗Öng nhanh v├á Tiß║┐t kiß╗çm ─æiß╗çn PCIe (ASPM)\n"+
			"┬╖ Gen2 sß║╜ tß╗▒ ─æß╗Öng mß╗ƒ kho├í khi ─æ─âng nhß║¡p\n\n"+
			"Nhß║¡t k├╜ chi tiß║┐t: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
		mbIconInfo)
}

func installDrivers() {
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	svcRunning := func(name string) bool {
		out, _ := hxcore.RunOut("sc.exe", "query", name)
		return strings.Contains(out, "RUNNING")
	}
	tsApp := hxcore.ThrottleStopAppRunning()
	for _, d := range []struct{ name, file string }{
		{"ThrottleStop", "ThrottleStop.sys"},
		{"WinRing0_1_2_0", "WinRing0x64.sys"},
	} {
		dst := filepath.Join(sysDir, d.file)
		if tsApp {
			fmt.Printf("  Ph├ít hiß╗çn phß║ºn mß╗üm ThrottleStop ─æang chß║íy, t├íi sß╗¡ dß╗Ñng driver %s (kh├┤ng ghi ─æ├¿/kh├┤ng xo├í)\n", d.name)
			continue
		}
		if svcRunning(d.name) {
			fmt.Printf("  %s ─æang chß║íy, bß╗Å qua ghi ─æ├¿ (giß╗» nguy├¬n trß║íng th├íi)\n", d.name)
			continue
		}
		hxcore.RunOut("sc.exe", "stop", d.name)
		pdDir := filepath.Join(os.Getenv("ProgramData"), "40HXUnlock", "drivers")
		os.MkdirAll(pdDir, 0o755)
		copyEmbedTo(filepath.Join(pdDir, d.file), d.file)
		if err := copyEmbedTo(dst, d.file); err != nil {
			if _, statErr := os.Stat(dst); statErr != nil {
				fmt.Printf("  [!] Sao ch├⌐p %s thß║Ñt bß║íi: %v\n", d.file, err)
				continue
			}
		} else {
			fmt.Printf("  ─É├ú sao ch├⌐p %s\n", d.file)
		}
		ensureService(d.name, d.file)
	}
	fmt.Println("  File driver Gen2 ─æ├ú sß║╡n s├áng (demand), ─æ─âng nhß║¡p sß║╜ ─æ╞░ß╗úc t├íc vß╗Ñ SYSTEM nß║íp v├á tß╗▒ dß╗ìn dß║╣p")
}

// ensureService: Σ╗àµ│¿σåî(µêûµ¢┤µû░)Θ⌐▒σè¿µ£ìσèí, Σ╕ìσ£¿µ¡ñσñäσèáΦ╜╜πÇé
// σ«ëΦúàΘÿ╢µ«╡σèáΦ╜╜ 40hx_bridge(µÿáσ░ä GPU BAR0)Σ╝ÜΣ╕Äµ¡úσ£¿Φ┐ÉΦíîτÜä nvlddmkm Σ║ëτö¿τí¼Σ╗╢,
// σ«₧µ╡ïσ»╝Φç┤ 40HX Φ«╛σñçµèÑ code19 / σÉÄτ╗¡σÉ»σè¿σ╝éσ╕╕πÇéσèáΦ╜╜µÄ¿Φ┐ƒσê░ΘçìσÉ»σÉÄτÖ╗σ╜òµù╢τÜä -gen2πÇé
// v2.4.6 σà│Θö«Σ┐«σñì(τñ╛σî║ #1/#2 µá╣σ¢á):
//
//	Θ⌐▒σè¿µ£ìσèíµ│¿σåîΣ╕║ start=demand(µëïσè¿), Θ£Çσ£¿τÖ╗σ╜òσÉÄτö▒ -gen2 µïëΦ╡╖πÇé
//	ΦÇî -gen2 Φ╡░ Run Θö«Σ╗ÑµÖ«ΘÇÜτö¿µê╖µ¥âΘÖÉΦ┐ÉΦíî ΓåÆ sc start Θ£ÇΦªüτ«íτÉåσæÿ ΓåÆ
//	"[SC] StartService: OpenService σñ▒Φ┤Ñ 5: µïÆτ╗¥Φ«┐Θù«" ΓåÆ Θ⌐▒σè¿µ░╕Φ┐£Φ╡╖Σ╕ìµ¥Ñ
//	ΓåÆ Gen2 µ░╕Φ┐£σñ▒Φ┤Ñ(τö¿µê╖τÄ░Φ▒í: τ«ùσè¢ΦºúΘöü OK Σ╜å Gen2 Γ£ù)πÇé
//	µ¡úΦºú = Σ┐¥µîü demand(Σ╕ìµö╣µêÉ auto! Φ»ªΦºüΣ╕ï), σ╣╢µèè -gen2 τÜäµëºΦíîµ¥âΘÖÉσìçσê░
//	SYSTEM: µ│¿σåî SYSTEM Φ«íσêÆΣ╗╗σèí(τÖ╗σ╜òµù╢ΦºªσÅæ + σ╗╢Φ┐ƒ 30s)Φ╖æ -gen2 -silent,
//	µùóΣ╕ìΘ£ÇΦªü UAC σ╝╣τ¬ù, σÅêΣ┐¥τòÖ"τÖ╗σ╜òσÉÄµëìσèáΦ╜╜Θ⌐▒σè¿"τÜäσ«ëσà¿µù╢σ║ÅπÇé
//
// Σ╕║Σ╗ÇΣ╣êΣ╕ìµö╣µêÉ start=auto: type=kernel auto Θ⌐▒σè¿σ£¿σ╝Çµ£║µù⌐µ£ƒτö▒ SCM σèáΦ╜╜,
// Σ╝ÜΣ╕ÄΘÜÅσÉÄσê¥σºïσîûτÜä nvlddmkm Σ║ëτö¿ GPU BAR0 ΓÇö σÄåσÅ▓Σ╕èσ«₧µ╡ïσ»╝Φç┤ 40HX µèÑ
// code19 / Windows σÉ»σè¿σ╝éσ╕╕Φ┐¢σ«ëσà¿µ¿íσ╝ÅπÇédemand + τÖ╗σ╜òσÉÄσèáΦ╜╜µÿ»τ╗ÅΦ┐çΘ¬îΦ»üτÜäµù╢σ║ÅπÇé
func ensureService(name string, sysFile string) {
	bin := fmt.Sprintf("\\SystemRoot\\System32\\drivers\\%s", sysFile)
	// σê¢σ╗║(σ╖▓σ¡ÿσ£¿Σ╝Üσñ▒Φ┤Ñ, σ┐╜τòÑ); σÉ»σè¿τ▒╗σ₧ï demand ΓÇö τö▒ SYSTEM Σ╗╗σèíτÖ╗σ╜òσÉÄµïëΦ╡╖
	hxcore.RunOut("sc.exe", "create", name, "type=", "kernel", "start=", "demand", "binPath=", bin)
	out, err := hxcore.RunOut("sc.exe", "query", name)
	if err != nil || !strings.Contains(out, "STATE") {
		fmt.Printf("  [!] µ│¿σåîµ£ìσèí %s σñ▒Φ┤Ñ: %s\n", name, strings.TrimSpace(out))
		return
	}
	// τ║áµ¡úΦó½σ«ëσà¿Φ╜»Σ╗╢/τ¡ûτòÑµö╣ΘöÖτÜäσÉ»σè¿τ▒╗σ₧ï(Disabled Σ╝Üσ»╝Φç┤ Gen2 µ░╕Φ┐£µïëΣ╕ìΦ╡╖)πÇé
	// σÉ»σè¿τ▒╗σ₧ïσ£¿ sc qc, Σ╕ìσ£¿ query; τè╢µÇü(STOPPED/RUNNING)σ£¿ queryπÇé
	start := "demand"
	if qc, qerr := hxcore.RunOut("sc.exe", "qc", name); qerr == nil {
		qcu := strings.ToUpper(qc)
		switch {
		case strings.Contains(qcu, "DISABLED"):
			hxcore.RunOut("sc.exe", "config", name, "start=", "demand")
			start = "demand (ban ─æß║ºu bß╗ï ─æß╗òi DISABLED, ─æ├ú sß╗¡a lß║íi)"
		case strings.Contains(qcu, "AUTO_START"):
			start = "auto (ch├║ ├╜: n├¬n l├á demand)"
		}
	}
	stateS := "?"
	switch {
	case strings.Contains(out, "RUNNING"):
		stateS = "RUNNING"
	case strings.Contains(out, "STOPPED"):
		stateS = "STOPPED"
	}
	fmt.Printf("  Dß╗ïch vß╗Ñ %s ─æ├ú ─æ─âng k├╜ (%s, %s), sau khi ─æ─âng nhß║¡p sß║╜ do t├íc vß╗Ñ SYSTEM nß║íp\n", name, start, stateS)
}

func setupBootEntry() error {
	// Bß╗Å qua nß║┐u mß╗Ñc "40HX Unlock" ─æ├ú tß╗ôn tß║íi
	if out, _ := hxcore.RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc) {
		fmt.Println("    Mß╗Ñc khß╗ƒi ─æß╗Öng ─æ├ú tß╗ôn tß║íi, bß╗Å qua")
		return nil
	}
	// 1. copy {bootmgr} l├ám mß║½u
	out, err := hxcore.RunOut("bcdedit.exe", "/copy", "{bootmgr}", "/d", bootDesc)
	if err != nil {
		return fmt.Errorf("bcdedit copy: %v", err)
	}
	re := regexp.MustCompile(`\{([0-9a-fA-F-]{36})\}`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return errors.New("Kh├┤ng thß╗â ph├ón t├¡ch ─æß║ºu ra bcdedit: " + out)
	}
	guid := m[1]
	cleanup := func() { hxcore.RunOut("bcdedit.exe", "/delete", "{"+guid+"}", "/f") }

	// 2. T├¼m k├╜ tß╗▒ ß╗ò ─æ─⌐a ESP (mountvol)
	esp := hxcore.MountESP()
	if esp == "" {
		cleanup()
		return errors.New("Kh├┤ng thß╗â gß║»n kß║┐t ph├ón v├╣ng ESP")
	}
	defer hxcore.UnmountESP(esp)

	// 3. set device + path
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{"+guid+"}", "device", "partition="+esp+":"); err != nil {
		cleanup()
		return err
	}
	path := efiDir + "\\" + efiFile // \EFI\40HX\40HXUNLK.EFI
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{"+guid+"}", "path", path); err != nil {
		cleanup()
		return err
	}
	// 4. displayorder addfirst
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{fwbootmgr}", "displayorder", "{"+guid+"}", "/addfirst"); err != nil {
		cleanup()
		return err
	}
	fmt.Printf("    Mß╗Ñc khß╗ƒi ─æß╗Öng %s ─æ├ú ─æ╞░ß╗úc ─æß║╖t ╞░u ti├¬n ─æß║ºu ti├¬n\n", guid)
	return nil
}

func setRunKey() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("  [!] Kh├┤ng thß╗â lß║Ñy ─æ╞░ß╗¥ng dß║½n file exe:", err)
		return
	}
	abs, _ := filepath.Abs(exe)
	val := fmt.Sprintf("\"%s\" -gen2 -silent", abs)
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		k, _, err = registry.CreateKey(registry.CURRENT_USER,
			`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	}
	if err != nil {
		fmt.Println("  [!] Ghi kh├│a Run thß║Ñt bß║íi:", err)
		return
	}
	defer k.Close()
	if err := k.SetStringValue("40HXGen2", val); err != nil {
		fmt.Println("  [!] Thiß║┐t lß║¡p kh├│a Run thß║Ñt bß║íi:", err)
		return
	}
	fmt.Println("  ─É├ú ghi kh├│a Run Gen2 (HKCU, nß║íp tß║ím thß╗¥i khi ─æ─âng nhß║¡p; t├íc vß╗Ñ SYSTEM l├á k├¬nh ch├¡nh thß╗⌐c): " + abs)
}

func setupGen2Task() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Kh├┤ng thß╗â lß║Ñy ─æ╞░ß╗¥ng dß║½n file exe: %v", err)
	}
	abs, _ := filepath.Abs(exe)
	tn := gen2TaskName
	var lastErr string
	for attempt := 1; attempt <= 3; attempt++ {
		out, cerr := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
			"/tr", fmt.Sprintf("\"%s\" -gen2 -silent -guard", abs),
			"/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:30", "/f")
		if cerr == nil {
			fmt.Println("  T├íc vß╗Ñ Gen2 ─æ├ú ─æ─âng k├╜ (SYSTEM, ─æß╗Ö trß╗à ─æ─âng nhß║¡p 30s, chß║íy ngß║ºm): " + abs)
			return nil
		}
		lastErr = strings.TrimSpace(out)
		if attempt < 3 {
			fmt.Printf("  [!] ─É─âng k├╜ t├íc vß╗Ñ thß║Ñt bß║íi (lß║ºn %d), ─æang thß╗¡ lß║íi... (%s)\n", attempt, lastErr)
			time.Sleep(800 * time.Millisecond)
		}
	}
	return fmt.Errorf("Tß║ío t├íc vß╗Ñ tß╗▒ ─æß╗Öng Gen2 thß║Ñt bß║íi (─æ├ú thß╗¡ lß║íi): %s\n      C├│ thß╗â thß╗º c├┤ng: Chß║íy 40HXInstaller.exe -task vß╗¢i quyß╗ün Administrator", lastErr)
}

// ===================== Gen2 ΦºúΘöü (σÄƒτöƒ, µùá python) =====================

func gen2Main() {
	// σ╣éτ¡ë; -silent(τÖ╗σ╜òΦç¬σÉ»σè¿Φ░âτö¿)µù╢σà¿τ¿ïµùáτ¬ùσÅúΘ¥ÖΘ╗ÿ
	// v2.5: BYOVD (ThrottleStop + WinRing0): Kh├┤ng cß║ºn test-signing, tß╗▒ ─æß╗Öng dß╗ìn dß║╣p khi ho├án tß║Ñt

	// v2.6.0: ─É╞ín phi├¬n duy nhß║Ñt: Tr├ính xung ─æß╗Öt khi t├íc vß╗Ñ SYSTEM, Run key hoß║╖c -gen2 thß╗º c├┤ng chß║íy ─æß╗ông thß╗¥i,
	// tr├ính hai tiß║┐n tr├¼nh c├╣ng sc start driver hoß║╖c tranh chß║Ñp BAR0 g├óy lß╗ùi trß║íng th├íi.
	// ─Éß║╖t ß╗ƒ ─æß║ºu: Kh├┤ng lß║Ñy ─æ╞░ß╗úc mutex th├¼ tho├ít ngay, tuyß╗çt ─æß╗æi kh├┤ng tß║úi driver.
	owned, release := gen2AcquireSingleInstance()
	if !owned {
		gen2Succeeded = true
		fmt.Println("[Gen2] Mß╗Öt tiß║┐n tr├¼nh Gen2 kh├íc ─æang chß║íy, bß╗Å qua (bß║úo vß╗ç ─æ╞ín phi├¬n)")
		_ = hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
			StatusCode: hxcore.StatusGen2Skipped,
			ErrorCode:  "ANOTHER_INSTANCE_RUNNING",
			Details: []string{
				"ΓÅ¡∩╕Å Bß╗Å qua: Mß╗Öt tiß║┐n tr├¼nh Gen2 kh├íc ─æang chß║íy (bß║úo vß╗ç ─æ╞ín phi├¬n, tr├ính xung ─æß╗Öt driver)",
			},
		})
		return
	}
	defer release()

	// v2.6.0: Bß║úo vß╗ç thß╗¥i gian: ─Éß╗úi nvlddmkm v├áo trß║íng th├íi RUNNING tr╞░ß╗¢c khi can thiß╗çp GPU.
	// Tr├ính thao t├íc tr╞░ß╗¢c khi driver nv sß║╡n s├áng v├¼ khi driver khß╗ƒi ─æß╗Öng c├│ thß╗â reset link PCIe hoß║╖c ghi ─æ├¿ thanh ghi GPU.
	// Tr├ính xung ─æß╗Öt vß╗¢i qu├í tr├¼nh khß╗ƒi ─æß╗Öng driver (thay thß║┐ delay cß╗æ ─æß╗ïnh 30s).
	waitForNvDriver(60 * time.Second)

	// Chß║íy to├án bß╗Ö chu tr├¼nh truy cß║¡p phß║ºn cß╗⌐ng v├á huß║Ñn luyß╗çn PCIe trong DriverSession kh├⌐p k├¡n (RAII)
	// Tß╗▒ ─æß╗Öng giß║úi ph├│ng handle v├á dß╗ìn sß║ích driver BYOVD khi kß║┐t th├║c
	err := hxcore.RunScopedBus(true, func(bus hxcore.HardwareBus) error {
		// ─Éß╗ïnh vß╗ï GPU ─æ╞░ß╗úc hß╗ù trß╗ú (40HX/30HX), kh├┤ng cß╗æ ─æß╗ïnh BDF
		var gpuBDF uint32
		var gpuProfile hxcore.GPUProfile
		gpuFound := false
		for attempt := 1; attempt <= 3; attempt++ {
			gpuBDF, gpuProfile, gpuFound = hxcore.FindGPUPCIWithBus(bus)
			if gpuFound {
				break
			}
			if attempt < 3 {
				fmt.Printf("[Gen2] Ch╞░a ─æß╗ïnh vß╗ï ─æ╞░ß╗úc GPU ─æ╞░ß╗úc hß╗ù trß╗ú, thß╗¡ lß║íi sau 2s (%d/3)...\n", attempt)
				time.Sleep(2 * time.Second)
			}
		}
		if !gpuFound {
			fmt.Println("[Gen2] Kh├┤ng thß╗â ─æß╗ïnh vß╗ï GPU ─æ╞░ß╗úc hß╗ù trß╗ú (40HX/30HX). Vui l├▓ng gß╗¡i file log.")
			gen2StatusFail("Kh├┤ng t├¼m thß║Ñy GPU ─æ╞░ß╗úc hß╗ù trß╗ú tr├¬n bus PCI")
			gen2Notify("Kh├┤ng t├¼m thß║Ñy GPU ─æ╞░ß╗úc hß╗ù trß╗ú tr├¬n bus PCI.\nVui l├▓ng kiß╗âm tra lß║íi card v├á driver.")
			return nil
		}

		if gpuProfile.HasSafePL0 {
			ensureGspSilent()
		}

		targetGen := uint32(2)
		if hasArg("-gen3") || hasArg("-gen3-30hx") || hasArg("-force-root-gen3") {
			if gpuProfile.MaxSupportedGen >= 3 {
				targetGen = 3
			} else {
				fmt.Printf("[Gen] Profile %s giß╗¢i hß║ín phß║ºn cß╗⌐ng tß╗æi ─æa Gen%d (eFuse lock), tß╗▒ ─æß╗Öng chuyß╗ân vß╗ü chß║┐ ─æß╗Ö Gen%d\n", gpuProfile.Name, gpuProfile.MaxSupportedGen, gpuProfile.MaxSupportedGen)
				targetGen = gpuProfile.MaxSupportedGen
			}
		}

		gpuBus := (gpuBDF >> 8) & 0xFF
		fmt.Printf("[Gen%d] %s tß║íi %02x:%02x.%x\n", targetGen, gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7)
		cur := bus.LinkSpeed(gpuBDF)
		fmt.Printf("[Gen%d] B─âng th├┤ng hiß╗çn tß║íi: Gen%d\n", targetGen, cur)
		// v2.6.0: Ghi nhß║¡n thanh ghi PCIe gß╗æc (LNKCAP/LNKCTL/LNKCTL2) ─æß╗â chß║⌐n ─æo├ín
		if cap := bus.PcieCap(gpuBDF); cap != 0 {
			rd := func(off uint32) uint32 {
				v, _ := bus.ReadPCIConfig(gpuBDF, cap+off)
				return v
			}
			fmt.Printf("[Gen%d] LNKCAP=0x%08X LNKCTL=0x%08X LNKCTL2=0x%08X (Mß╗Ñc ti├¬u Gen%d)\n",
				targetGen, rd(0x0C), rd(0x10), rd(0x30), rd(0x30)&0xF)
		}
		if cur >= targetGen {
			gen2Succeeded = true
			fmt.Printf("[Gen%d] ─É├ú ─æß║ít Gen%d, kh├┤ng cß║ºn thao t├íc th├¬m.\n", targetGen, cur)
			stContract := hxcore.StatusContract{
				StatusCode:   hxcore.StatusGen2Success,
				SpeedCurrent: cur,
				WidthCurrent: bus.LinkWidth(gpuBDF),
				TLSTarget:    targetGen,
				ErrorCode:    "NONE",
				Details: []string{
					fmt.Sprintf("Kß║┐t luß║¡n: Γ£à Gen%d kh├┤ng cß║ºn thao t├íc: B─âng th├┤ng hiß╗çn tß║íi ─æ├ú l├á Gen%d", targetGen, cur),
					fmt.Sprintf("Quyß╗ün thß╗▒c thi: %s", map[bool]string{true: "Quß║ún trß╗ï vi├¬n (Admin)/SYSTEM", false: "Ng╞░ß╗¥i d├╣ng th╞░ß╗¥ng (Bß╗ï hß║ín chß║┐)"}[isAdmin()]),
					fmt.Sprintf("Vß╗ï tr├¡ %s: %02x:%02x.%x", gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
					fmt.Sprintf("─æ├ú ─æß║ít mß╗Ñc ti├¬u Gen%d th├ánh c├┤ng", targetGen),
				},
			}
			_ = hxcore.WriteStructuredGen2Status(stContract)
			gen2Notify(fmt.Sprintf("PCIe ─æ├ú ─æß║ít Gen%d, kh├┤ng cß║ºn thao t├íc th├¬m.", cur))
			return nil
		}

		// PCIe Capability gate (Ph├¡a GPU + Ph├¡a Root Port)
		root := bus.FindRootPort(gpuBus)
		if root == 0xFFFFFFFF {
			fmt.Printf("[Gen%d] Kh├┤ng t├¼m thß║Ñy root port, d├╣ng GPU retrain dß╗▒ ph├▓ng\n", targetGen)
		} else {
			fmt.Printf("[Gen%d] root port = 00:%02x.%x\n", targetGen, (root>>3)&0x1F, root&7)
		}
		gpuMax := bus.PcieMaxSpeed(gpuBDF)
		rootMax := uint32(0)
		if root != 0xFFFFFFFF {
			rootMax = bus.PcieMaxSpeed(root)
		} else {
			rootMax = gpuMax
		}
		fmt.Printf("[Gen%d] Khß║ú n─âng PCIe: GPU Max=Gen%d, Root Max=Gen%d, Giß╗¢i hß║ín Profile=Gen%d\n", targetGen, gpuMax, rootMax, gpuProfile.MaxSupportedGen)
		allowTarget := hxcore.LinkTargetAllowed(gpuMax, rootMax, gpuProfile.MaxSupportedGen, targetGen)
		forceRoot := hasArg("-force-root-gen2") || hasArg("-force-root-gen3") || hasArg("-gen2-30hx") || hasArg("-gen3-30hx") || gpuProfile.DeviceID == 0x2189

		if !allowTarget {
			if forceRoot && gpuProfile.DeviceID == 0x2189 && rootMax >= targetGen {
				fmt.Printf("[Gen%d] Card ─æß╗ô hoß║í b├ío LNKCAP Gen%d, k├¡ch hoß║ít chß║┐ ─æß╗Ö huß║Ñn luyß╗çn Gen%d: Root Port (Max=Gen%d) khß╗ƒi tß║ío huß║Ñn luyß╗çn Gen%d\n", targetGen, gpuMax, targetGen, rootMax, targetGen)
			} else if forceRoot && gpuProfile.DeviceID == 0x2189 && targetGen == 3 && rootMax < 3 {
				fmt.Printf("[Gen3][!] Phß║ºn cß╗⌐ng Root Port chß╗ë hß╗ù trß╗ú Gen%d (< Gen3), hß║í xuß╗æng Gen%d ─æß╗â thß╗¡ nghiß╗çm\n", rootMax, rootMax)
				targetGen = rootMax
				if targetGen < 2 {
					msg := fmt.Sprintf("Root Port chß╗ë hß╗ù trß╗ú Gen%d, kh├┤ng thß╗â ─æß║ít Gen2/Gen3", rootMax)
					fmt.Printf("[Gen3][!] %s, an to├án dß╗½ng lß║íi.\n", msg)
					gen2StatusFail(msg)
					return nil
				}
			} else {
				width := bus.LinkWidth(gpuBDF)
				tls := uint32(0)
				if cap := bus.PcieCap(gpuBDF); cap != 0 {
					if v, err := bus.ReadPCIConfig(gpuBDF, cap+0x30); err == nil {
						tls = v & 0xF
					}
				}
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: GPU Device ID: %04X:%04X\n", targetGen, gpuProfile.VendorID, gpuProfile.DeviceID)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: GPU Family: %s\n", targetGen, gpuProfile.Family)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: GPU Max Link Speed: Gen%d\n", targetGen, gpuMax)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Root Port Max Link Speed: Gen%d\n", targetGen, rootMax)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Current Link Speed: Gen%d\n", targetGen, cur)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Current Width: x%d\n", targetGen, width)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Target TLS: Gen%d\n", targetGen, tls)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Mutation: skipped\n", targetGen)
				fmt.Printf("[Gen%d] Chß║⌐n ─æo├ín: Reason: endpoint advertises Gen%d (< Gen%d)\n", targetGen, gpuMax, targetGen)

				msg := fmt.Sprintf("Phß║ºn cß╗⌐ng hoß║╖c Profile kh├┤ng hß╗ù trß╗ú Gen%d (GPU Max=%d, Root Max=%d, Cap=%d)", targetGen, gpuMax, rootMax, gpuProfile.MaxSupportedGen)
				fmt.Printf("[Gen%d][!] %s, an to├án dß╗½ng lß║íi.\n", targetGen, msg)
				gen2StatusFail(msg)
				gen2Notify(fmt.Sprintf("%s Li├¬n kß║┐t phß║ºn cß╗⌐ng PCIe kh├┤ng hß╗ù trß╗ú Gen%d (chß╗ë Gen%d), an to├án dß╗½ng lß║íi.\nRoot Port Max=Gen%d\nCß║ºn kiß╗âm tra thiß║┐t lß║¡p BIOS khe cß║»m bo mß║ích chß╗º, riser/d├óy nß╗æi hoß║╖c giß╗¢i hß║ín VBIOS/Strap.\nNß║┐u muß╗æn Root Port ├⌐p huß║Ñn luyß╗çn lß║íi, th├¬m tham sß╗æ: -force-root-gen%d", gpuProfile.Name, targetGen, gpuMax, rootMax, targetGen))
				return nil
			}
		}

		negotiator := hxcore.NewLinkNegotiator(bus)
		allowStage2 := hasArg("-hard") || (gpuProfile.DeviceID != 0x2189 && gen2AutoHardEnabled())

		fmt.Printf("[Gen%d] Khß╗ƒi chß║íy LinkNegotiator cho %s (DEV_%04X)...\n", targetGen, gpuProfile.Name, gpuProfile.DeviceID)
		res, err := negotiator.Negotiate(gpuBDF, gpuProfile, root, targetGen, allowStage2)
		if err != nil {
			fmt.Printf("[Gen%d][!] Lß╗ùi th╞░╞íng l╞░ß╗úng link: %v\n", targetGen, err)
			gen2StatusFail(fmt.Sprintf("Lß╗ùi th╞░╞íng l╞░ß╗úng link: %v", err))
			return nil
		}

		gen2Succeeded = res.Success
		if res.Success {
			deleteGen2Retry()
		} else if hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
			scheduleGen2Retry(retryDepth())
		}

		fmt.Printf("[Gen%d] %s\n", res.TargetGen, res.Verdict)

		statusCode := hxcore.StatusGen1Stuck
		if res.Success {
			statusCode = hxcore.StatusGen2Success
		}
		details := []string{
			fmt.Sprintf("Kß║┐t luß║¡n: %s", res.Verdict),
			fmt.Sprintf("Quyß╗ün thß╗▒c thi: %s", map[bool]string{true: "Quß║ún trß╗ï vi├¬n (Admin)/SYSTEM", false: "Ng╞░ß╗¥i d├╣ng th╞░ß╗¥ng (Bß╗ï hß║ín chß║┐)"}[isAdmin()]),
			fmt.Sprintf("%s Vß╗ï tr├¡: %02x:%02x.%x", gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
			fmt.Sprintf("Root Port: %02x:%02x.%x", (root>>8)&0xFF, (root>>3)&0x1F, root&7),
			fmt.Sprintf("B─âng th├┤ng: Hiß╗çn tß║íi Gen%d x%d / GPU TLS=Gen%d / Root TLS=Gen%d", res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS),
		}
		if gpuProfile.DeviceID == 0x2189 {
			details = append(details, "MRRS: 512B [─É├ú tß╗æi ╞░u]")
		}
		if res.Success {
			details = append(details, fmt.Sprintf("─æ├ú ─æß║ít mß╗Ñc ti├¬u Gen%d th├ánh c├┤ng", res.TargetGen))
		} else {
			details = append(details, fmt.Sprintf("ch╞░a ─æß║ít mß╗Ñc ti├¬u Gen%d", res.TargetGen))
		}

		stContract := hxcore.StatusContract{
			StatusCode:   statusCode,
			SpeedCurrent: res.CurrentSpeed,
			WidthCurrent: res.CurrentWidth,
			TLSTarget:    res.TargetGen,
			ErrorCode:    "NONE",
			Details:      details,
		}
		if err := hxcore.WriteStructuredGen2Status(stContract); err != nil {
			fmt.Printf("[Gen%d] Ghi file trß║íng th├íi thß║Ñt bß║íi: %v\n", res.TargetGen, err)
		}

		if !hasArg("-silent") && !hasArg("-y") {
			icon := uint(mbIconInfo)
			txt := fmt.Sprintf("B─âng th├┤ng PCIe: Hiß╗çn tß║íi Gen%d x%d (GPU TLS=Gen%d, Root TLS=Gen%d)\n", res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS)
			if res.Success {
				if res.CurrentSpeed < res.TargetGen {
					txt += fmt.Sprintf("\nGen1 l├║c nh├án rß╗ùi l├á tiß║┐t kiß╗çm ─æiß╗çn b├¼nh th╞░ß╗¥ng; h├úy chß║íy GPU-Z Render Test hoß║╖c tß║úi 3D/CUDA ─æß╗â x├íc nhß║¡n Gen%d.", res.TargetGen)
				}
				txt += fmt.Sprintf("\n=== Mß╗₧ KHO├ü GEN%d TH├ÇNH C├öNG ===", res.TargetGen)
			} else {
				txt += fmt.Sprintf("\nVß║½n ß╗ƒ Gen%d, ch╞░a ─æß║ít Gen%d. Xem %s v├á kiß╗âm tra HVCI, riser/khe PCIe, BIOS; sau ─æ├│ thß╗¡ lß║íi.", res.CurrentSpeed, res.TargetGen, filepath.Join(os.TempDir(), "40HX_installer.log"))
				icon = mbIconError
			}
			msgbox(fmt.Sprintf("%s Gen%d", gpuProfile.Name, res.TargetGen), txt, icon)
		}
		return nil
	})

	if err != nil {
		if !isAdmin() {
			fmt.Println("[Gen2] Lß╗ùi tß║úi driver v├á hiß╗çn tß║íi kh├┤ng c├│ quyß╗ün Admin: Chuyß╗ân cho SYSTEM task, tho├ít im lß║╖ng:", err)
			gen2StatusFail("Driver ch╞░a ─æ╞░ß╗úc tß║úi, hiß╗çn tß║íi quyß╗ün hß║ín bß╗ï giß╗¢i hß║ín (do task SYSTEM xß╗¡ l├╜)")
			return
		}
		fmt.Println("[Gen2] Lß╗ùi khß╗ƒi ─æß╗Öng driver:", err)
		gen2StatusFail("Driver kh├┤ng chß║íy ─æ╞░ß╗úc: " + err.Error())
		gen2Notify("Lß╗ùi khß╗ƒi ─æß╗Öng driver kernel.\nNguy├¬n nh├ón: Antivirus chß║╖n WinRing0/ThrottleStop hoß║╖c xung ─æß╗Öt phß║ºn mß╗üm.\nVui l├▓ng chß║íy vß╗¢i quyß╗ün Administrator.")
		return
	}
}

// probe30HX: Chß║⌐n ─æo├ín chß╗ë ─æß╗ìc thanh ghi BAR0 MMIO link v├á PHY CMP 30HX (TU116)
func probe30HX() {
	fmt.Println("=== Chß║⌐n ─æo├ín chß╗ë ─æß╗ìc thanh ghi BAR0 MMIO CMP 30HX (TU116) ===")
	err := hxcore.RunScopedBus(true, func(bus hxcore.HardwareBus) error {
		gpuBDF, gpuProfile, gpuFound := hxcore.FindGPUPCIWithBus(bus)
		if !gpuFound {
			fmt.Println("[!] Kh├┤ng ─æß╗ïnh vß╗ï ─æ╞░ß╗úc card ─æß╗ô hoß║í hß╗ù trß╗ú (30HX/40HX) tr├¬n bus PCI")
			return nil
		}
		gpuBus := (gpuBDF >> 8) & 0xFF
		fmt.Printf("[Probe] Card ─æß╗ô hoß║í: %s (DEV_%04X) tß║íi %02x:%02x.%x\n", gpuProfile.Name, gpuProfile.DeviceID, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7)

		bar0raw, err := bus.ReadPCIConfig(gpuBDF, 0x10)
		if err != nil || bar0raw == 0 || bar0raw == 0xFFFFFFFF {
			fmt.Printf("[!] ─Éß╗ìc BAR0 bß║Ñt th╞░ß╗¥ng: 0x%08X (err=%v)\n", bar0raw, err)
			return nil
		}
		bar0Phys := uint64(bar0raw & 0xFFFFFFF0)
		fmt.Printf("[Probe] PCI BAR0 (0x10) = 0x%08X (─Éß╗ïa chß╗ë vß║¡t l├╜ gß╗æc: 0x%08X)\n", bar0raw, bar0Phys)

		boot0, berr := bus.ReadMMIO(bar0Phys + 0x0)
		if berr != nil {
			fmt.Printf("[!] ─Éß╗ìc BOOT_0 thß║Ñt bß║íi: %v\n", berr)
			return nil
		}
		fmt.Printf("[Probe] NV_PMC_BOOT_0 (BAR0+0x00000) = 0x%08X\n", boot0)

		regs := []struct {
			off  uint64
			name string
		}{
			{0x00088084, "LNKCAP (NV_XVE 0x84)"},
			{0x000880A4, "LNKCAP2 (NV_XVE 0xA4)"},
			{0x000880A8, "LNKCTL2 (NV_XVE 0xA8)"},
			{0x000880F0, "NV_XVE_0xF0"},
			{0x0008841C, "NV_XVE_PRIV_MISC_1"},
			{0x00088700, "NV_XVE_0x700"},
			{0x00088708, "NV_XVE_0x708"},
			{0x0008870C, "NV_XVE_0x70C"},
			{0x00088714, "NV_XVE_0x714"},
			{0x00088720, "NV_XVE_0x720"},
			{0x0008872C, "NV_XVE_OVR (0x72C)"},
			{0x0008C040, "NV_XVE_LINK_CONFIG_0"},
			{0x0008C1C0, "NV_XVE_PL_LINK_RATE"},
			{0x0008C2C0, "NV_XVE_CYA_0"},
			{0x0008C4B0, "PHY_LANE0_SPEED (2.5G/5G)"},
			{0x0008C4B4, "PHY_LANE1_SPEED"},
			{0x0008C4B8, "PHY_LANE2_SPEED"},
			{0x0008C4BC, "PHY_LANE3_SPEED"},
		}

		fmt.Println("\n[Probe] ===== Gi├í trß╗ï thß╗▒c ─æo thanh ghi link PCIe v├á PHY BAR0 =====")
		for _, r := range regs {
			val, rerr := bus.ReadMMIO(bar0Phys + r.off)
			if rerr != nil {
				fmt.Printf("  0x%06X (%-26s): ─Éß╗ìc thß║Ñt bß║íi (%v)\n", r.off, r.name, rerr)
			} else {
				extra := ""
				if r.off == 0x0008C2C0 {
					if (val & (1 << 2)) != 0 {
						extra = " [bit2=1 DIS_G2 bß║¡t -> kh├│a Gen2!]"
					} else {
						extra = " [bit2=0 DIS_G2 tß║»t -> cho ph├⌐p Gen2]"
					}
				}
				fmt.Printf("  0x%06X (%-26s): 0x%08X%s\n", r.off, r.name, val, extra)
			}
		}
		fmt.Println("======================================================")
		return nil
	})
	if err != nil {
		fmt.Printf("[!] Lß╗ùi khß╗ƒi tß║ío phi├¬n driver chß║⌐n ─æo├ín: %v\n", err)
	}
}

// ---------- v3.0.1: Daemon th╞░ß╗¥ng tr├║ (khi ch├¡nh s├ích driver=resident, khß╗ƒi chß║íy tß╗½ t├íc vß╗Ñ ─æ─âng nhß║¡p -guard) ----------
const gen2GuardInterval = 1 * time.Minute

func residentGuard() {
	fmt.Println("[Gi├ím s├ít] Khß╗ƒi ─æß╗Öng daemon th╞░ß╗¥ng tr├║: Mß╗ùi 1 ph├║t kiß╗âm tra mß╗Ñc ti├¬u Gen2 (TLS), tß╗▒ ─æß╗Öng huß║Ñn luyß╗çn lß║íi nß║┐u mß║Ñt cß║Ñu h├¼nh (TLS<2); dß╗½ng khi ─æ─âng xuß║Ñt hoß║╖c t├íc vß╗Ñ kß║┐t th├║c.")
	for {
		time.Sleep(gen2GuardInterval)
		st := hxcore.ReadUnlockStateV2(0, 0)
		if st.Speed >= 2 || st.TLS >= 2 {
			continue // Mß╗Ñc ti├¬u vß║½n duy tr├¼: Hiß╗çn tß║íi Gen2 hoß║╖c hß║í tß╗æc khi rß║únh l├á b├¼nh th╞░ß╗¥ng
		}
		fmt.Println("[Gi├ím s├ít] Ph├ít hiß╗çn Speed<2 && TLS<2 ΓÇö Mß║Ñt cß║Ñu h├¼nh mß╗ƒ kho├í Gen2, tß╗▒ ─æß╗Öng mß╗ƒ kho├í lß║íi...")
		gen2Main()
	}
}

// ---------- v2.6.0: Gen2 Φç¬σè¿ΘçìΦ»ò + Stage2 Φç¬σè¿σ¢₧ΘÇÇσ╝Çσà│ + τ¡ûτòÑΘàìτ╜« ----------

// retryDepth: σ╜ôσëìΦç¬σè¿ΘçìΦ»òµ╖▒σ║ª(-retrydepth=N, 0=τÖ╗σ╜òΣ╗╗σèíΘªûµ¼íµëºΦíî)
func retryDepth() int {
	for _, a := range os.Args {
		if strings.HasPrefix(a, "-retrydepth=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "-retrydepth=")); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// gen2AutoHardEnabled: Stage2(Link Disable + PnP µüóσñì)Φç¬σè¿µëºΦíîσ╝Çσà│, Θ╗ÿΦ«ñσ╝ÇπÇé
// σà│Θù¡: reg add HKLM\SOFTWARE\40HXUnlock /v Gen2AutoHard /t REG_DWORD /d 0 /f
// (40HX µÿ»σö»Σ╕Çµÿ╛τñ║σìíπÇüΣ╕ìσ╕îµ£¢τÖ╗σ╜òσÉÄΘô╛Φ╖»τ₧¼µû¡µò░τºÆΘ╗æσ▒ÅτÜäτö¿µê╖σÅ»σà│)
func gen2AutoHardEnabled() bool {
	return hxcore.ConfigInt("Gen2AutoHard", 1) != 0
}

// scheduleGen2Retry: σñ▒Φ┤ÑσÉÄσ«ëµÄÆΣ╕Çµ¼íµÇºΦç¬σè¿ΘçìΦ»ò(SYSTEM, Θ¥ÖΘ╗ÿ, Θ╗ÿΦ«ñ 15 σêåΘÆƒσÉÄ)πÇé
// Φªåτ¢û"σ╝Çµ£║σÉÄΘ⌐▒σè¿/GSP σ░▒τ╗¬µàó""Θô╛Φ╖»τè╢µÇüµü░σÑ╜σìíΣ╜Å"τ¡ëµù╢σ║Åτ▒╗σñ▒Φ┤Ñ(τñ╛σî║ #12);
// depth Σ╕║σ╖▓ΘçìΦ»òµ¼íµò░, Φ╢àσç║τ¡ûτòÑΘóäτ«ù(Gen2RetryCount)σì│Σ╕ìσåìµÄÆ; µêÉσèƒΦ╖»σ╛ä deleteGen2RetryπÇé
func scheduleGen2Retry(depth int) {
	count, interval := hxcore.Gen2RetryPolicy()
	if depth >= count {
		fmt.Printf("[Gen2] Φç¬σè¿ΘçìΦ»òΘóäτ«ùσ╖▓τö¿σ«î(%d/%d), τ¡ëΣ╕ïµ¼íτÖ╗σ╜òσåìΦ»ò\n", depth, count)
		return
	}
	t := time.Now().Add(time.Duration(interval) * time.Minute)
	if t.Day() != time.Now().Day() {
		fmt.Println("[Gen2] Gß║ºn nß╗¡a ─æ├¬m, bß╗Å qua lß╗ïch tr├¼nh thß╗¡ lß║íi lß║ºn n├áy (t├íc vß╗Ñ once qua ng├áy kh├┤ng ─æ├íng tin cß║¡y)")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	abs, _ := filepath.Abs(exe)
	out, err := hxcore.RunOut("schtasks.exe", "/create", "/tn", gen2RetryTask,
		"/tr", fmt.Sprintf("\"%s\" -gen2 -silent -retrydepth=%d", abs, depth+1),
		"/sc", "once", "/st", t.Format("15:04"), "/ru", "SYSTEM", "/f")
	if err != nil {
		fmt.Printf("[Gen2] Tß║ío t├íc vß╗Ñ thß╗¡ lß║íi thß║Ñt bß║íi (kh├┤ng ß║únh h╞░ß╗ƒng mß╗ƒ kho├í): %s\n", strings.TrimSpace(out))
		return
	}
	fmt.Printf("[Gen2] ─É├ú l├¬n lß╗ïch tß╗▒ ─æß╗Öng thß╗¡ lß║íi sau %d ph├║t (%d/%d, t├íc vß╗Ñ %s)\n", interval, depth+1, count, gen2RetryTask)
}

// deleteGen2Retry: Xo├í t├íc vß╗Ñ thß╗¡ lß║íi nß║┐u c├│ sau khi ─æß║ít Gen2 th├ánh c├┤ng
func deleteGen2Retry() {
	hxcore.RunOut("schtasks.exe", "/delete", "/tn", gen2RetryTask, "/f")
}

// gen2AcquireSingleInstance: v2.6.0 Bß║úo vß╗ç ─æ╞ín phi├¬n bß║ún
func gen2AcquireSingleInstance() (bool, func()) {
	name, _ := windows.UTF16PtrFromString("Global\\40HXGen2SingleInstance")
	h, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		fmt.Println("[Gen2] Tß║ío mutex ─æ╞ín phi├¬n bß║ún thß║Ñt bß║íi, bß╗Å qua:", err)
		return true, func() {}
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(windows.Handle(h))
		return false, nil
	}
	return true, func() {
		_ = windows.ReleaseMutex(windows.Handle(h))
		_ = windows.CloseHandle(windows.Handle(h))
	}
}

// waitForNvDriver: ─Éß╗úi driver nvlddmkm chuyß╗ân sang trß║íng th├íi RUNNING
func waitForNvDriver(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		out, _ := hxcore.RunOut("sc.exe", "query", "nvlddmkm")
		if strings.Contains(out, "does not exist") || strings.Contains(out, "ch╞░a c├ái ─æß║╖t") ||
			strings.Contains(out, "1060") {
			fmt.Println("[Gen2] Kh├┤ng ph├ít hiß╗çn dß╗ïch vß╗Ñ nvlddmkm, bß╗Å qua chß╗¥ ─æß╗úi v├á tiß║┐p tß╗Ñc mß╗ƒ kho├í")
			return true
		}
		if strings.Contains(out, "RUNNING") {
			return true
		}
		if time.Now().After(deadline) {
			fmt.Printf("[Gen2] nvlddmkm kh├┤ng chuyß╗ân sang RUNNING trong v├▓ng %s (xem log), tiß║┐p tß╗Ñc mß╗ƒ kho├í\n", timeout)
			return false
		}
		fmt.Println("[Gen2] ─Éang chß╗¥ nvlddmkm sß║╡n s├áng...")
		time.Sleep(2 * time.Second)
	}
}

// gen2Notify: Th├┤ng b├ío lß╗ùi; chß║┐ ─æß╗Ö im lß║╖ng kh├┤ng hiß╗çn hß╗Öp thoß║íi
func gen2Notify(txt string) {
	if !hasArg("-silent") && !hasArg("-y") {
		msgbox("40HX Gen2", txt, mbIconError)
	}
}

// gen2StatusFail: Ghi l├╜ do Gen2 kh├┤ng thß╗▒c thi/thß║Ñt bß║íi v├áo file trß║íng th├íi
func gen2StatusFail(reason string) {
	ident := map[bool]string{true: "Quß║ún trß╗ï vi├¬n/SYSTEM", false: "Ng╞░ß╗¥i d├╣ng th╞░ß╗¥ng (Bß╗ï hß║ín chß║┐)"}[isAdmin()]
	code := hxcore.StatusDrvFail
	errCode := "DRV_BLOCKED"
	low := strings.ToLower(reason)
	if strings.Contains(low, "pci") || strings.Contains(low, "kh├┤ng t├¼m thß║Ñy") || strings.Contains(low, "ch╞░a ─æß╗ïnh vß╗ï") {
		code = hxcore.StatusNoGPU
		errCode = "GPU_NOT_FOUND"
	}
	_ = hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
		StatusCode: code,
		ErrorCode:  errCode,
		Details: []string{
			"Γ¥î Gen2 Ch╞░a thß╗▒c thi: " + reason,
			"Quyß╗ün thß╗▒c thi: " + ident,
		},
	})
}

// ===================== Gß╗í c├ái ─æß║╖t / Trß║íng th├íi =====================

func uninstall() {
	if !isAdmin() {
		fmt.Println("[!] Cß║ºn quyß╗ün quß║ún trß╗ï vi├¬n.")
		msgbox("Bß╗Ö c├ái ─æß║╖t 40HX", "Cß║ºn quyß╗ün quß║ún trß╗ï vi├¬n.\nVui l├▓ng nhß║Ñp chuß╗Öt phß║úi v├áo ch╞░╞íng tr├¼nh -> Chß╗ìn Run as administrator.", mbIconError)
		return
	}
	if lockOnce(`Local\40HXUninstaller_v1`) == nil {
		msgbox("Bß╗Ö c├ái ─æß║╖t 40HX", "Ch╞░╞íng tr├¼nh gß╗í c├ái ─æß║╖t ─æang chß║íy, vui l├▓ng kh├┤ng nhß║Ñp tr├╣ng lß║╖p.", mbIconInfo)
		return
	}
	fmt.Println("=== Gß╗í c├ái ─æß║╖t mß╗ƒ kho├í 40HX (v3.0.0 Cß║Ñp th├ánh phß║ºn) ===")
	fmt.Print("[1/8] Xo├í t├íc vß╗Ñ lß╗ïch tr├¼nh ... ")
	if rem := hxcore.UninstallTasks(); len(rem) > 0 {
		fmt.Println("Ho├án th├ánh")
	} else {
		fmt.Println("Kh├┤ng t├¼m thß║Ñy (bß╗Å qua)")
	}
	fmt.Print("[2/8] Xo├í kho├í Run tß╗▒ khß╗ƒi ─æß╗Öng Gen2 ... ")
	hxcore.UninstallRunKey()
	fmt.Println("Ho├án th├ánh")
	fmt.Print("[3/8] Xo├í mß╗Ñc khß╗ƒi ─æß╗Öng firmware '40HX Unlock' ... ")
	if hxcore.UninstallBootEntry() {
		fmt.Println("Ho├án th├ánh")
	} else {
		fmt.Println("Kh├┤ng t├¼m thß║Ñy (c├│ thß╗â ─æ├ú ─æ╞░ß╗úc gß╗í bß╗Å)")
	}
	fmt.Print("[4/8] Xo├í EFI mß╗ƒ kho├í trong ESP ... ")
	if hxcore.UninstallEspEfi() {
		fmt.Println("Ho├án th├ánh")
	} else {
		fmt.Println("Kh├┤ng t├¼m thß║Ñy/bß╗Å qua")
	}
	fmt.Println("[5/8] Dß╗½ng v├á xo├í dß╗ïch vß╗Ñ driver...")
	hxcore.UninstallDriverServices()
	fmt.Println("[6/8] Xo├í file driver...")
	hxcore.UninstallDriverFiles()
	fmt.Print("[6.5/8] Xo├í EnableGpuFirmware (kh├┤i phß╗Ñc GSP vß╗ü tß║»t mß║╖c ─æß╗ïnh) ... ")
	if hxcore.UninstallGspKey() {
		fmt.Println("Ho├án th├ánh")
	} else {
		fmt.Println("Kh├┤ng t├¼m thß║Ñy (bß╗Å qua)")
	}
	fmt.Print("[6.6/8] Dß╗ìn dß║╣p ProgramData + kho├í ch├¡nh s├ích ... ")
	hxcore.UninstallProgramData()
	fmt.Println("Ho├án th├ánh")
	fmt.Print("[6.7/8] Dß╗ìn dß║╣p mß╗Ñc loß║íi trß╗½ Defender ... ")
	if err := hxcore.RemoveDefenderExclusions(); err != nil {
		fmt.Println("Ch╞░a thß╗▒c thi (c├│ thß╗â bß╗Å qua):", err)
	} else {
		fmt.Println("Ho├án th├ánh")
	}
	fmt.Println("[7/8] Kiß╗âm tra t├án d╞░...")
	left := hxcore.CheckLeftover()
	fmt.Println()
	fmt.Println("Gß╗í c├ái ─æß║╖t ho├án tß║Ñt. Khuyß║┐n nghß╗ï khß╗ƒi ─æß╗Öng lß║íi m├íy t├¡nh.")
	fmt.Println("  L╞░u ├╜: Thiß║┐t lß║¡p nguß╗ôn khi c├ái ─æß║╖t (Fast Startup/ASPM) ─æ╞░ß╗úc giß╗» nguy├¬n ΓÇö c├ích kh├┤i phß╗Ñc xem README ┬º2.4.")
	icon := uint(mbIconInfo)
	txt := "Gß╗í c├ái ─æß║╖t ho├án tß║Ñt.\nKhuyß║┐n nghß╗ï khß╗ƒi ─æß╗Öng lß║íi m├íy t├¡nh.\n\nL╞░u ├╜: Thiß║┐t lß║¡p nguß╗ôn khi c├ái ─æß║╖t (Fast Startup/ASPM)\n─æ╞░ß╗úc giß╗» nguy├¬n theo sß╗ƒ th├¡ch nguß╗ôn ΓÇö c├ích kh├┤i phß╗Ñc xem README ┬º2.4.\n"
	if len(left) > 0 {
		icon = mbIconError
		txt += "\nVß║½n c├▓n t├án d╞░:\n" + strings.Join(left, "\n")
	}
	txt += "\nNhß║¡t k├╜ chi tiß║┐t: " + filepath.Join(os.TempDir(), "40HX_installer.log")
	msgbox("Bß╗Ö c├ái ─æß║╖t 40HX", txt, icon)
}

func status() {
	prof, gpuOK := hxcore.FindGPUWithProfile()
	cardName := "40HX"
	if gpuOK {
		cardName = prof.Name
	}
	fmt.Printf("=== Trß║íng th├íi mß╗ƒ kho├í %s ===\n", cardName)
	sb := hxcore.SecureBootOn()
	ts := hxcore.TestSigningOn()
	fmt.Printf("Ph├ít hiß╗çn GPU %s: %v\n", cardName, gpuOK)
	fmt.Printf("Secure Boot: %v\n", sb)
	fmt.Printf("Testsigning: %v\n", ts)

	gs := false
	if gpuOK && !prof.FirmwareUnlock {
		fmt.Printf("Phß║ºn cß╗⌐ng GPU: %s (%s, DEV_%04X)\n", prof.Name, prof.Family, prof.DeviceID)
		fmt.Println("Firmware GSP: Kh├┤ng cß║ºn (Kiß║┐n tr├║c TU116 kh├┤ng phß╗Ñ thuß╗Öc GSP-RM)")
	} else {
		gs = hxcore.GspEnabled()
		fmt.Printf("Bß║¡t GSP (EnableGpuFirmware=1): %v\n", gs)
		if sub, adapter, fw := hxcore.GspDiag(); sub != "" {
			fmt.Printf("  Kho├í GSP: Class\\%s (fw=%d)\n", sub, fw)
			fmt.Printf("  AdapterString: %s\n", adapter)
		} else {
			fmt.Println("  [!] " + adapter)
		}
	}
	dep := hxcore.InspectGen2Drivers()
	if !hxcore.Gen2DriversDeployedOnce() {
		fmt.Println("Driver Gen2: Ch╞░a tß╗½ng triß╗ân khai ΓÇö chß║íy bß╗Ö c├ái ─æß║╖t v├á khß╗ƒi ─æß╗Öng lß║íi ─æß╗â c├│ hiß╗çu lß╗▒c")
	} else {
		for _, d := range dep {
			svcS := "Ch╞░a ─æ─âng k├╜"
			if d.SvcReg {
				svcS = d.SvcStart
				if d.SvcRunning {
					svcS += "/─Éang chß║íy"
				}
			}
			fmt.Printf("Driver Gen2 %-16s Nguß╗ôn sao l╞░u=%v  System32=%s  Dß╗ïch vß╗Ñ=%s\n",
				d.File, map[bool]string{true: "OK", false: "Kh├┤ng"}[d.BackupOK], d.SysState.String(), svcS)
		}
	}
	if ex, err := hxcore.DefenderExclusionsPresent(); err != nil {
		fmt.Println("Loß║íi trß╗½ Defender: Truy vß║Ñn thß║Ñt bß║íi (" + err.Error() + ")")
	} else if ex {
		fmt.Println("Loß║íi trß╗½ Defender: ─É├ú th├¬m danh s├ích trß║»ng (OK)")
	} else {
		fmt.Println("Loß║íi trß╗½ Defender: C├▓n thiß║┐u ΓÇö phß║ºn mß╗üm diß╗çt virus c├│ thß╗â xo├í nhß║ºm driver")
	}
	st := hxcore.ReadUnlockStateV2(5, 800)
	tsRun := st.TSOK
	winringRun := st.WinRingOK
	speed := st.Speed
	ss0 := st.SS0
	ss0ok := st.SS0OK

	if gpuOK && !prof.FirmwareUnlock {
		fmt.Printf("WinRing0 (Truy cß║¡p PCI Config): %v\n", winringRun)
		fmt.Printf("ThrottleStop: %v (30HX kh├┤ng cß║ºn driver n├áy)\n", tsRun)
		if winringRun {
			fmt.Printf("B─âng th├┤ng PCIe: Gen%d\n", speed)
		} else {
			fmt.Println("Driver ch╞░a chß║íy (cß║ºn WinRing0 ─æß╗â kiß╗âm tra v├á huß║Ñn luyß╗çn lß║íi PCIe)")
		}
	} else {
		fmt.Printf("ThrottleStop: %v\n", tsRun)
		fmt.Printf("WinRing0: %v\n", winringRun)
		if tsRun && winringRun {
			fmt.Printf("B─âng th├┤ng PCIe: Gen%d\n", speed)
			if ss0ok {
				fmt.Printf("SS0 (Hashrate): 0x%08x %s\n", ss0, map[bool]string{true: "(─É├ú mß╗ƒ kho├í)", false: "(Kho├í)"}[st.Unlocked])
			}
		} else {
			fmt.Println("Driver ch╞░a chß║íy (sß║╡n s├áng kiß╗âm tra Gen2/trß║íng th├íi sau khi c├ái ─æß║╖t)")
		}
	}

	diag := []string{}
	if !gpuOK {
		diag = append(diag, fmt.Sprintf("┬╖ Kh├┤ng ph├ít hiß╗çn %s ΓÇöΓÇö Vui l├▓ng kiß╗âm tra cß║»m card v├á c├ái driver", cardName))
	}
	if gpuOK && !prof.FirmwareUnlock {
		if !winringRun {
			diag = append(diag, "┬╖ Driver WinRing0 ch╞░a chß║íy: Chß║íy thß╗º c├┤ng 40HXInstaller.exe -gen2 hoß║╖c khß╗ƒi ─æß╗Öng lß║íi hß╗ç thß╗æng")
		} else {
			diag = append(diag, fmt.Sprintf("┬╖ B─âng th├┤ng PCIe hiß╗çn tß║íi: Gen%d", speed))
			if speed < 2 {
				diag = append(diag, "┬╖ Link vß║½n l├á Gen1: Cß║ºn kiß╗âm tra tß╗æc ─æß╗Ö khe cß║»m BIOS mainboard, d├óy c├íp riser hoß║╖c giß╗¢i hß║ín VBIOS/Strap")
			}
		}
	} else {
		if sb {
			diag = append(diag, "┬╖ Secure Boot ─æang bß║¡t: Cß║ºn v├áo BIOS tß║»t ─æi, nß║┐u kh├┤ng EFI mß╗ƒ kho├í sß║╜ bß╗ï tß╗½ chß╗æi")
		}
		if ts {
			diag = append(diag, "┬╖ Testsigning ─æang bß║¡t ΓÇö v2.5 kh├┤ng cß║ºn thiß║┐t, c├│ thß╗â chß║íy bcdedit /set testsigning off ─æß╗â tß║»t")
		}
		if !gs {
			diag = append(diag, "┬╖ GSP ch╞░a bß║¡t: C├│ thß╗â ─æen m├án h├¼nh sau khi mß╗ƒ kho├í. Chß║íy bß╗Ö c├ái ─æß║╖t (tß╗▒ ─æß╗Öng bß║¡t EnableGpuFirmware=1)")
		}
		if !tsRun || !winringRun {
			diag = append(diag, "┬╖ Driver ch╞░a chß║íy: Sß║╜ tß╗▒ khß╗ƒi ─æß╗Öng sau khi ─æ─âng nhß║¡p; hoß║╖c chß║íy thß╗º c├┤ng 40HXInstaller.exe -gen2")
		}
		if tsRun && winringRun {
			if !ss0ok {
				diag = append(diag, "┬╖ Driver ─æ├ú chß║íy nh╞░ng kh├┤ng ─æß╗ìc ─æ╞░ß╗úc thanh ghi hashrate (bß║Ñt th╞░ß╗¥ng)")
			} else if ss0 == 0x88888888 {
				diag = append(diag, fmt.Sprintf("┬╖ SS0=0x%08x: Hashrate ─æ├ú mß╗ƒ kho├í! PCIe Gen%d", ss0, speed))
			} else {
				diag = append(diag, fmt.Sprintf("┬╖ SS0=0x%08x: Hashrate vß║½n kho├í ΓÇöΓÇö Khi khß╗ƒi ─æß╗Öng lß║íi 40HX Unlock EFI ch╞░a thß╗▒c thi th├ánh c├┤ng", ss0))
				efiDiag := hxcore.AnalyzeEfiLog()
				if efiDiag != "" {
					diag = append(diag, efiDiag)
				}
			}
		}
	}

	msg := fmt.Sprintf("Trß║íng th├íi mß╗ƒ kho├í %s\n========================\n", cardName)
	msg += fmt.Sprintf("GPU %s: %v    Secure Boot: %v\n", cardName, map[bool]string{true: "Γ£ô", false: "Γ£ù"}[gpuOK], map[bool]string{true: "Bß║¡t!", false: "Tß║»t (OK)"}[sb])
	if gpuOK && !prof.FirmwareUnlock {
		msg += fmt.Sprintf("WinRing0: %v    PCIe: Gen%d\n", map[bool]string{true: "Γ£ô", false: "Γ£ù"}[winringRun], speed)
	} else {
		msg += fmt.Sprintf("Testsigning: %v    GSP: %v\n", map[bool]string{true: "Γ£ô", false: "Γ£ù"}[ts], map[bool]string{true: "Γ£ô", false: "Γ£ù"}[gs])
		msg += fmt.Sprintf("ThrottleStop: %v  WinRing0: %v\n", map[bool]string{true: "Γ£ô", false: "Γ£ù"}[tsRun], map[bool]string{true: "Γ£ô", false: "Γ£ù"}[winringRun])
		if tsRun && winringRun {
			msg += fmt.Sprintf("PCIe: Gen%d    SS0: 0x%08x\n", speed, ss0)
		}
	}
	msg += "\nChß║⌐n ─æo├ín:\n" + strings.Join(diag, "\n")
	if len(diag) == 0 {
		msg += "┬╖ Mß╗ìi thß╗⌐ hoß║ít ─æß╗Öng b├¼nh th╞░ß╗¥ng"
	}
	msg += "\n\nNhß║¡t k├╜ chi tiß║┐t: " + filepath.Join(os.TempDir(), "40HX_installer.log")
	msgbox(fmt.Sprintf("Trß║íng th├íi %s", cardName), msg, mbIconInfo)
	fmt.Println("=== Kß║┐t th├║c trß║íng th├íi ===")
}

func pause() {
	// GUI τëê: µùáΘ£Çµîë Enter; Φ╛ôσç║σ╖▓σàÑµùÑσ┐ù, Σ║ñΣ║Æµö╢σ░╛τö¿µ╢êµü»µíå
}
