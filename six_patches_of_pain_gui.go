//go:generate goversioninfo -icon=rinnegan.ico
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	//"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
    //"log"
	"net/http"
    //"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/cheggaaa/pb/v3"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
	//"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
	//"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/data/binding"
    //"fyne.io/fyne/v2/driver/desktop"
    //"fyne.io/fyne/v2/theme"
    //"fyne.io/fyne/v2/cmd/fyne_demo/tutorials"
    //"fyne.io/fyne/v2/cmd/fyne_settings/settings"

)

// DATA folder for data files
var DATA = "data"

// Xdelta3Exe Windows binary
var Xdelta3Exe = "data/xdelta3.exe"

// Xdelta3 Linux binary
var Xdelta3 = "xdelta3"

// Xdelta Mac binary
var Xdelta = "xdelta"

// CurrentVersion current version to see if a newer version exists
var CurrentVersion = "data/current_version"

// GitRepository git repo to download new releases from
var GitRepositoryFile = "data/git_repository"

// DefaultGitRepository default git repository to download new releases from
var GitRepository = "https://api.github.com/repos/NicholasMoser/SCON4-Releases/releases"

// PatchFile the patch file to be downloaded
var PatchFile = "data/patch"

// GNT4ISOPath path of the GNT4 ISO if it's not in the current directory
var GNT4ISOPath = "data/gnt4_iso_path"

// GNT4ISO default name of the GNT4 iso if the user downloads it
var GNT4ISO = "data/GNT4.iso"

// argISOPath path of the GNT4 ISO given as argument
var argISOPath string

// WindowsExecutableName the name of the Windows executable
var WindowsExecutableName = "Six-Patches-Of-Pain-GUI.exe"

// LinuxExecutableName the name of the Mac and Linux executable
var LinuxExecutableName = "Six-Patches-Of-Pain-GUI"

// ExecutableName the name of the executable
var ExecutableName string

// The main gui application
var a = app.NewWithID("six-patches-of-pain")
var version = "1.2.0"
var w = a.NewWindow("Six Patches of Pain v" + version)

func main() {
	verifyIntegrity()
	//ov := a.NewWindow("Select version")

    w.Resize(fyne.NewSize(800, 400))

    var data []string //listVersions()

    gnt4Iso := GNT4ISO//getGNT4ISO()

    list := widget.NewList(
        func() int {
            return len(data)
        },
        func() fyne.CanvasObject {
            return widget.NewLabel("template")
        },
        func(i widget.ListItemID, o fyne.CanvasObject) {
            o.(*widget.Label).SetText(data[i])
        })
	list.Resize(fyne.NewSize(100,300))

	repoString := binding.NewString()
	repoString.Set(GitRepository)
	repo := widget.NewEntry()
	repo.Bind(repoString)
	repo.Resize(fyne.NewSize(550,40))
	repo.Enable()
	repo.Move(fyne.NewPos(10,120))
    gnt4label := widget.NewLabel(gnt4Iso)
	gnt4label.Move(fyne.NewPos(10,40))
    upgrade_latest := widget.NewButton("Upgrade latest", func() {
		if isGNT4(gnt4Iso) {
			r, _ := repoString.Get()
			newVersion := downloadNewVersion(r)
			outputIso := fmt.Sprintf("SCON4-%s.iso", newVersion)
			patchGNT4(gnt4Iso, outputIso)
			setCurrentVersion(newVersion)
			setGitRepo(r)
		}
    })
    fileDiag := dialog.NewFileOpen(func(uri fyne.URIReadCloser, e error) {
        if uri != nil {
            gnt4Iso = uri.URI().Path()
            gnt4label.SetText(gnt4Iso)
            gnt4label.Refresh()
			setGNT4ISOPath(gnt4Iso)
        }
    }, w)
    gntIsoButton := widget.NewButton("Specify GNT4 iso path", func() {
        fileDiag.Show()
    })
    list.OnSelected = func(id widget.ListItemID) {
		if isGNT4(gnt4Iso) {
			r, _ := repoString.Get()
			newVersion := downloadVersion(r,id)
			outputIso := fmt.Sprintf("SCON4-%s.iso", newVersion)
			patchGNT4(gnt4Iso, outputIso)
			setGitRepo(r)
		}
    }
    asd := widget.NewPopUp(list, w.Canvas())
	asd.Resize(fyne.NewSize(170,200))
	asd.Move(fyne.NewPos(150,20))
    oldPatchButton := widget.NewButton("Get list of old patches", func() {
		r, _ := repoString.Get()
        data = listVersions(r)
        list.CreateItem()
		asd.Show()
    })
    v := container.NewVBox(upgrade_latest, gntIsoButton, oldPatchButton, widget.NewLabel("Repository URL:"))
	vl:= container.NewWithoutLayout(repo, gnt4label)
    h := container.NewHBox(v,vl)

    w.SetContent(h)

    w.ShowAndRun()
}

// Verify the integrity of the auto-updater and required files.
func verifyIntegrity() {
	// Check that xdelta3 exists
	if runtime.GOOS == "windows" {
		ExecutableName = WindowsExecutableName
		if !exists(Xdelta3Exe) {
			// Make sure that the current working directory is at the exe
			// This may not be true when dragging and dropping an ISO in Windows
			if strings.HasSuffix(os.Args[0], WindowsExecutableName) {
				dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
				check(err)
				os.Chdir(dir)
			}
			if !exists(Xdelta3Exe) {
				fmt.Println("Unable to find xdelta3.exe in the data folder.")
				fmt.Println("Please make sure you extracted the entire zip file, not just Six-Patches-Of-Pain.exe")
				fmt.Println()
				fmt.Println("After following the above instructions, if you still encounter issues:")
				fmt.Println("Please verify that there is a folder named data with a file named xdelta3.exe")
				fmt.Println("If you do not see it, redownload and extract Six Patches of Pain.")
				fmt.Println("If you still don't see xdelta3.exe it may be an issue with your antivirus.")
				fail()
			}
		}
	} else if runtime.GOOS == "darwin" {
		// Create the data directory if it doesn't already exist
		if !exists(DATA) {
			err := os.Mkdir(DATA, 0755)
			check(err)
		}
		ExecutableName = LinuxExecutableName
		if !isCommandAvailable(Xdelta) {
			fmt.Println("Unable to find xdelta, please install xdelta.")
			fail()
		}
	} else {
		// Create the data directory if it doesn't already exist
		if !exists(DATA) {
			err := os.Mkdir(DATA, 0755)
			check(err)
		}
		if !isCommandAvailable(Xdelta3) {
			fmt.Println("Unable to find xdelta3, please install xdelta3.")
			fail()
		}
		ExecutableName = LinuxExecutableName
	}
	if !exists(GitRepositoryFile) {
		d1 := []byte(GitRepository)
		err := ioutil.WriteFile(GitRepositoryFile, d1, 0644)
		check(err)
	}
	GitRepository = strings.TrimSpace(readFile(GitRepositoryFile))
	// If iso path is not set, set it to the default. If set, but different from argument, reset if saveConfig arg is set
	if !exists(GNT4ISOPath) {
		d1 := []byte(GNT4ISO)
		err := ioutil.WriteFile(GNT4ISOPath, d1, 0644)
		check(err)
	}
	GNT4ISO = strings.TrimSpace(readFile(GNT4ISOPath))
	// Delete any existing patch files, since they may be corrupted/old
	if exists(PatchFile) {
		os.Remove(PatchFile)
	}
}

// Download a new release if it exists and return the version name.
func downloadNewVersion(repo string) string {
	// Get the latest release
	resp, err := http.Get(repo)
	check(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Unable to access releases for %s\nStatus code: %s", repo, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	check(err)
	var f interface{}
	err2 := json.Unmarshal(body, &f)
	check(err2)
	releases := f.([]interface{})
	if len(releases) == 0 {
		fmt.Println("No releases found at " + repo)
		fail()
	}
	latestRelease := releases[0].(map[string]interface{})
	// Stop if the latest release has already been patched locally
	latestVersion := latestRelease["name"].(string)
	if exists(CurrentVersion) {
		currentVersion := readFile(CurrentVersion)
		if currentVersion == latestVersion {
			fmt.Println("Already on latest SCON4 version: " + latestVersion)
			fmt.Println("If you wish to re-download the latest version,")
			fmt.Println("please delete the file named current_version in the data folder.")
			fail()
		}
	}
	// Download the patch
	assets := latestRelease["assets"].([]interface{})
	if len(assets) == 0 {
		fmt.Println("No assets found in latest release for " + repo)
		fail()
	} else if len(assets) > 1 {
		fmt.Println("Too many assets found in latest release for " + repo)
		fail()
	}
	downloadURL := assets[0].(map[string]interface{})["browser_download_url"].(string)
	fmt.Println("There is a new version of SCON4 available: " + latestVersion)
	fmt.Println("Downloading: " + latestVersion)
	download(downloadURL, PatchFile)
	return latestVersion
}

func downloadVersion(repo string, i int) string {
	// Get a specific release
	resp, err := http.Get(repo)
	check(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Unable to access releases for %s\nStatus code: %s", repo, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	check(err)
	var f interface{}
	err2 := json.Unmarshal(body, &f)
	check(err2)
	releases := f.([]interface{})
	if len(releases) == 0 {
		fmt.Println("No releases found at " + repo)
		fail()
	}
	specificRelease := releases[i].(map[string]interface{})
	specificVersion := specificRelease["name"].(string)
	// Download the patch
	assets := specificRelease["assets"].([]interface{})
	if len(assets) == 0 {
		fmt.Println("No assets found in latest release for " + repo)
		fail()
	} else if len(assets) > 1 {
		fmt.Println("Too many assets found in latest release for " + repo)
		fail()
	}
	downloadURL := assets[0].(map[string]interface{})["browser_download_url"].(string)
	fmt.Println("Downloading: " + specificVersion)
	download(downloadURL, PatchFile)
	return specificVersion
}


func listVersions(repo string) []string {
	// Get a specific release
	resp, err := http.Get(repo)
	check(err)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Unable to access releases for %s\nStatus code: %s", repo, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	check(err)
	var f interface{}
	err2 := json.Unmarshal(body, &f)
	check(err2)
	releases := f.([]interface{})
	if len(releases) == 0 {
		fmt.Println("No releases found at " + repo)
		fail()
	}
    var versions []string
    for i:=0; i<len(releases); i++ {
        //fmt.Println(i,": ",releases[i].(map[string]interface{})["name"].(string))
        versions = append(versions, releases[i].(map[string]interface{})["name"].(string))
    }
    return versions
}

// Patches the given GNT4 ISO to the output SCON4 ISO path using the downloaded patch.
func patchGNT4(gnt4Iso string, scon4Iso string) {
	fmt.Println("Patching GNT4...")
	scon4Iso = path.Dir(gnt4Iso) + "/" + scon4Iso
	var xdelta string
	if runtime.GOOS == "windows" {
		xdelta = Xdelta3Exe
	} else if runtime.GOOS == "darwin" {
		xdelta = Xdelta
	} else {
		xdelta = Xdelta3
	}
	cmd := exec.Command(xdelta, "-f", "-d", "-s", gnt4Iso, PatchFile, scon4Iso)
	out, err := cmd.CombinedOutput()
	check(err)
	fmt.Printf("%s\n", out)
	if exists(scon4Iso) && getFileSize(scon4Iso) > 0 {
		isoFullPath, err := filepath.Abs(scon4Iso)
		check(err)
		fmt.Println("Patching complete. Saved to " + isoFullPath)
	}
}

// Returns whether or not the given file path is vanilla GNT4.
func isGNT4(filePath string) bool {
	if strings.HasSuffix(strings.ToLower(filePath), ".iso") {
		f, err := os.Open(filePath)
		check(err)
		data := make([]byte, 6)
		len, err := f.Read(data)
		check(err)
		f.Close()
		expected := []byte("G4NJDA")
		if reflect.DeepEqual(expected, data[:len]) {
			fmt.Println("Validating GNT4 ISO is not modified...")
			hashValue, err := hashFile(filePath)
			check(err)
			if hashValue == "60aefa3e" {
				// 60aefa3e is the hash for a good dump, but we currently use a "bad" dump instead.
				// The bad dump is superior as it pads with zeroes instead of random bytes.
				// Confirm the user is okay with modifying their good dump to be a bad dump.
				fmt.Println("\nThe vanilla ISO you provided must be modified in order to be used for this auto updater.")
				fmt.Println("Please press enter if you are okay with this ISO being modified.")
				fmt.Println("If you are not okay with this ISO being modified, please exit this application.")
				fmt.Println("\nFor more information, see the following information:")
				fmt.Println("https://github.com/NicholasMoser/Six-Patches-Of-Pain#why-does-it-say-my-vanilla-iso-needs-to-be-modified")
				fmt.Println("\nPress enter to continue...")
				var output string
				fmt.Scanln(&output)
				err = patchGoodDump(filePath)
				fmt.Println("\nISO has been modified and is now valid.")
				check(err)
				return true
			}
			return hashValue == "55ee8b1a"
		}
	}
	return false
}

// Patches a good dump of vanilla GNT4 to be the expected "bad" dump of GNT4
func patchGoodDump(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	// First write this weird four byte word to bi2.bin
	_, err = file.WriteAt([]byte{0x00, 0x52, 0x02, 0x02}, 0x500)
	if err != nil {
		return err
	}
	var zeroes [4096]byte
	// There are random padding bytes from 0x248104 to 0xC4F8000 (0xC2AFEFC bytes).
	// Replace them with zeroes by looping 49839 times. Then add 3836 extra zeroes.
	for i := 0; i < 49839; i++ {
		offset := 0x248104 + (i * 4096)
		_, err := file.WriteAt(zeroes[:], int64(offset))
		if err != nil {
			return err
		}
	}
	var moreZeroes [3836]byte
	_, err = file.WriteAt(moreZeroes[:], 0xC4F7104)
	if err != nil {
		return err
	}
	var evenMoreZeroes [11108]byte
	// There are random padding bytes from 0x4553001C - 0x45532B7F (0x2B63 bytes).
	// Just add 11108 zeroes directly.
	_, err = file.WriteAt(evenMoreZeroes[:], 0x4553001C)
	return err
}

// Download to a file path the file at the given url.
func download(url string, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Unable to download file, status: " + resp.Status)
	}
	bar := pb.Full.Start64(resp.ContentLength)
	defer bar.Finish()
	barReader := bar.NewProxyReader(resp.Body)
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, barReader)
	return err
}

func setGitRepo(url string) {
	data := []byte(url)
	err := ioutil.WriteFile(GitRepositoryFile, data, 0644)
	check(err)
}

//  Set the vanilla GNT4 ISO path to the vanilla GNT4 ISO path file.
func setGNT4ISOPath(filePath string) {
	data := []byte(filePath)
	err := ioutil.WriteFile(GNT4ISOPath, data, 0644)
	check(err)
}

// Set the new version to the current version file.
func setCurrentVersion(version string) {
	data := []byte(version)
	err := ioutil.WriteFile(CurrentVersion, data, 0644)
	check(err)
}

// Retrieves the CRC32 hash of a given file.
func hashFile(filePath string) (string, error) {
	var returnCRC32String string
	fileSize := getFileSize(filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return returnCRC32String, err
	}
	defer file.Close()
	bar := pb.Full.Start64(fileSize)
	defer bar.Finish()
	barReader := bar.NewProxyReader(file)
	tablePolynomial := crc32.MakeTable(crc32.IEEE)
	hash := crc32.New(tablePolynomial)
	if _, err := io.Copy(hash, barReader); err != nil {
		return returnCRC32String, err
	}
	hashInBytes := hash.Sum(nil)[:]
	returnCRC32String = hex.EncodeToString(hashInBytes)
	return returnCRC32String, nil
}

// Check if a command is available. Shamelessly borrowed from
// https://siongui.github.io/2018/03/16/go-check-if-command-exists/
func isCommandAvailable(name string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// Delete the patch file if it exists and exit with exit code 1.
func fail() {
	if exists(PatchFile) {
		os.Remove(PatchFile)
	}
	exit(1)
}

// Read a file to a string
func readFile(filePath string) string {
	content, err := ioutil.ReadFile(filePath)
	check(err)
	return string(content)
}

// Get the file size for a file (assumes the file exists)
func getFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	check(err)
	return fi.Size()
}

// Return whether or not the given path exists.
func exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// Query user to exit and exit with given code.
func exit(code int) {
	fmt.Println("\nPress enter to exit...")
	var output string
	fmt.Scanln(&output)
	os.Exit(code)
}

func check(e error) {
	if e != nil {
		//panic(e)
		dialog.NewError(e, w).Show()
	}
}
