//go:generate goversioninfo -icon=rinnegan.ico
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/cheggaaa/pb/v3"
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

// argGitRepository git repository given as argument to download new releases from
var argGitRepository string

// argSpecificVersion boolean that specifies if you want to select which version to download
var argSpecificVersion bool

// PatchFile the patch file to be downloaded
var PatchFile = "data/patch"

// GNT4ISOPath path of the GNT4 ISO if it's not in the current directory
var GNT4ISOPath = "data/gnt4_iso_path"

// GNT4ISO default name of the GNT4 iso if the user downloads it
var GNT4ISO = "data/GNT4.iso"

// argISOPath path of the GNT4 ISO given as argument
var argISOPath string

// WindowsExecutableName the name of the Windows executable
var WindowsExecutableName = "Six-Patches-Of-Pain.exe"

// LinuxExecutableName the name of the Mac and Linux executable
var LinuxExecutableName = "Six-Patches-Of-Pain"

// ExecutableName the name of the executable
var ExecutableName string

func main() {
	version := "1.2.0"
	fmt.Printf("Starting Six Patches of Pain %s....\n", version)
	fmt.Println()
	argParse()
	verifyIntegrity()
	gnt4Iso := getGNT4ISO()
    var newVersion string
    if argSpecificVersion {
        newVersion = downloadSpecificVersion()
    } else {
        newVersion = downloadNewVersion()
    }
	outputIso := fmt.Sprintf("SCON4-%s.iso", newVersion)
	patchGNT4(gnt4Iso, outputIso)
	setCurrentVersion(newVersion)
	if exists(PatchFile) {
		os.Remove(PatchFile)
	}
	exit(0)
}

// Parse the arguments
func argParse() {
	flag.StringVar(&argGitRepository,"r","","Specify git repository to download updates from as 'https://api.github.com/repos/{user}/{repository}/releases'")
	flag.StringVar(&argISOPath,"p","","Specify path of the GNT4 ISO")
  flag.BoolVar(&argSpecificVersion,"specific",false,"Select a specific version to download")
	flag.Parse()
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
    if argGitRepository != "" {
        GitRepository = argGitRepository
    }
    if argISOPath != "" {
        GNT4ISO = argISOPath
    }
	// If git repository is not set, set it to the default release repository. If set, but different from argument, reset if saveConfig arg is set
	if !exists(GitRepositoryFile) {
		d1 := []byte(GitRepository)
		err := ioutil.WriteFile(GitRepositoryFile, d1, 0644)
		check(err)
	}
	if argGitRepository != "" && readFile(GitRepositoryFile) != argGitRepository {
		d1 := []byte(GitRepository)
		err := ioutil.WriteFile(GitRepositoryFile, d1, 0644)
		check(err)
	}
	// If iso path is not set, set it to the default. If set, but different from argument, reset if saveConfig arg is set
	if !exists(GNT4ISOPath) {
		d1 := []byte(GNT4ISO)
		err := ioutil.WriteFile(GNT4ISOPath, d1, 0644)
		check(err)
	}
	if argISOPath != "" && readFile(GNT4ISOPath) != GNT4ISO {
		d1 := []byte(GNT4ISO)
		err := ioutil.WriteFile(GNT4ISOPath, d1, 0644)
		check(err)
	}
	// Delete any existing patch files, since they may be corrupted/old
	if exists(PatchFile) {
		os.Remove(PatchFile)
	}
}

// Retrieves the vanilla GNT4 iso to patch against.
func getGNT4ISO() string {
	// First, check if it was drag and dropped onto the executable or provided as an arg
	if len(os.Args) == 2 && !argSpecificVersion {
		var draggedPath = os.Args[1]
		if exists(draggedPath) {
			if isGNT4(draggedPath) {
				setGNT4ISOPath(draggedPath)
				return draggedPath
			}
			fmt.Println("Provided file is not a vanilla GNT4 ISO: " + draggedPath)
		} else {
			fmt.Println("Provided path is not valid: " + draggedPath)
		}
	}
	// Then look for if it was provided as a named arg
	isoPath := argISOPath
	if exists(isoPath) {
		return isoPath
	}
	// Then look for the ISO in GNT4_ISO_PATH
	if exists(GNT4ISOPath) {
		isoPath := readFile(GNT4ISOPath)
		if exists(isoPath) {
			return isoPath
		}
	}
	// If the ISO isn't found from the previous step, look for it recursively in the current directory
	gnt4Iso := ""
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isGNT4(path) {
			gnt4Iso = path
		}
		return nil
	})
	check(err)
	if gnt4Iso != "" {
		setGNT4ISOPath(gnt4Iso)
		return gnt4Iso
	}
	// Last resort, query the user for the location of the ISO
	for true {
		fmt.Printf("This updater requires a vanilla GNT4 ISO in order to auto-update.\n")
		fmt.Printf("Please do one of the following:\n")
		fmt.Printf("  - Exit this application and drag and drop your vanilla GNT4 ISO onto %s\n", ExecutableName)
		fmt.Printf("  - Enter the file path to your local copy of a vanilla GNT4 ISO\n")
		fmt.Printf("  - Move a vanilla GNT4 ISO to this folder and restart %s\n", ExecutableName)
		fmt.Printf("  - Enter a link to a download for a vanilla GNT4 ISO\n")
		fmt.Println()
		fmt.Print("Input: ")
		var input string
		fmt.Scanln(&input)
		if exists(input) {
			// Local file
			if isGNT4(input) {
				setGNT4ISOPath(input)
				return input
			}
			fmt.Printf("\nERROR: %s is not a clean vanilla GNT4 ISO\n\n", input)
		} else {
			// Download from interwebs
			err := download(input, GNT4ISO)
			if err != nil {
				fmt.Printf("Failed to download file with error: %s\n\n", err.Error())
				if exists(GNT4ISO) {
					os.Remove(GNT4ISO)
				}
			} else {
				if exists(GNT4ISO) {
					if isGNT4(GNT4ISO) {
						setGNT4ISOPath(GNT4ISO)
						return GNT4ISO
					}
					fmt.Printf("\nERROR: Downloaded file was not a vanilla GNT4 ISO.\n\n")
					os.Remove(GNT4ISO)
				}
			}
		}
	}
	return ""
}

// Download a new release if it exists and return the version name.
func downloadNewVersion() string {
	// Get the latest release
	repo := readFile(GitRepositoryFile)
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

// Specify which available version to download
func downloadSpecificVersion() string {
	// Get a specific release
	repo := readFile(GitRepositoryFile)
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
    for i:=0; i<len(releases); i++ {
        fmt.Println(i,": ",releases[i].(map[string]interface{})["name"].(string))
    }
    fmt.Print("Enter the number of the wished release: ")
	var input int
	fmt.Scanln(&input)
    if input >= len(releases) {
        input = len(releases) - 1
    } else if input < 0 {
        input = 0
    }
	specificRelease := releases[input].(map[string]interface{})
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

// Patches the given GNT4 ISO to the output SCON4 ISO path using the downloaded patch.
func patchGNT4(gnt4Iso string, scon4Iso string) {
	fmt.Println("Patching GNT4...")
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
		panic(e)
	}
}
