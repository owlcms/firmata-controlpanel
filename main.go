package main

import (
	"fmt"
	"image/color"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"

	"firmata-launcher/downloadUtils"
	"firmata-launcher/javacheck"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Masterminds/semver/v3"
)

var (
	owlcmsInstallDir          = getInstallDir()
	currentProcess            *exec.Cmd
	currentVersion            string // Add to track current version
	statusLabel               *widget.Label
	stopButton                *widget.Button
	versionContainer          *fyne.Container
	stopContainer             *fyne.Container
	singleOrMultiVersionLabel *widget.Label     // New label for single or multi version update
	downloadContainer         *fyne.Container   // New global to track the same container
	downloadsShown            bool              // New global to track whether downloads are shown
	urlLink                   *widget.Hyperlink // Add this new variable
)

func init() {
	javacheck.InitJavaCheck(owlcmsInstallDir)
}

type myTheme struct {
	fyne.Theme
}

func newMyTheme() *myTheme {
	return &myTheme{Theme: theme.LightTheme()}
}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameSuccess:
		// Darker green color (forest green)
		return color.RGBA{R: 34, G: 139, B: 34, A: 255}
	// case theme.ColorNameForegroundOnSuccess:
	// 	// Black text
	// 	return color.Black
	case theme.ColorNameBackground:
		return color.White
	case theme.ColorNameForeground:
		return color.Black
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 40}
	default:
		return m.Theme.Color(name, variant)
	}
}

func getInstallDir() string {
	switch downloadUtils.GetGoos() {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "firmata")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "firmata")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".local", "share", "firmata")
	default:
		return "./firmata"
	}
}

func checkJava(statusLabel *widget.Label) error {
	statusLabel.SetText("Checking for the Java language runtime.")
	statusLabel.Refresh()
	statusLabel.Show()
	stopButton.Hide()
	stopContainer.Show()
	versionContainer.Hide()
	downloadContainer.Hide()

	err := javacheck.CheckJava(statusLabel)
	if err != nil {
		statusLabel.SetText("Could not install a Java runtime.")
		statusLabel.Refresh()
		return err
	}

	statusLabel.Hide() // Hide the status label if Java check is successful
	return nil
}

func goBackToMainScreen() {
	stopButton.Hide()
	stopContainer.Hide()
	downloadContainer.Show()
	versionContainer.Show()
}

func computeVersionScrollHeight(numVersions int) float32 {
	minHeight := 0  // minimum height
	rowHeight := 50 // approximate height per row
	return float32(minHeight + (rowHeight * min(numVersions, 4)))
}

func removeAllVersions() {
	entries, err := os.ReadDir(owlcmsInstallDir)
	if err != nil {
		log.Printf("Failed to read firmata directory: %v\n", err)
		dialog.ShowError(fmt.Errorf("failed to read firmata directory: %w", err), fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			_, err := semver.NewVersion(entry.Name())
			if err == nil {
				dirPath := filepath.Join(owlcmsInstallDir, entry.Name())
				if err := os.RemoveAll(dirPath); err != nil {
					log.Printf("Failed to remove directory %s: %v\n", dirPath, err)
					dialog.ShowError(fmt.Errorf("failed to remove directory %s: %w", dirPath, err), fyne.CurrentApp().Driver().AllWindows()[0])
					return
				}
			}
		}
	}

	log.Println("All versions removed successfully")
	dialog.ShowInformation("Success", "All versions removed successfully", fyne.CurrentApp().Driver().AllWindows()[0])
	getAllInstalledVersions()
	updateTitle.ParseMarkdown("All Versions Removed.")
	downloadButtonTitle.SetText("Click here to install a version.")
	downloadButtonTitle.Refresh()
	updateTitle.Refresh()
	recomputeVersionList(fyne.CurrentApp().Driver().AllWindows()[0])
}

func uninstallAll() {
	dialog.ShowConfirm("Confirm Uninstall", "This will remove all the data and configurations currently stored and exit the program.\nIf you proceed, this cannot be undone. Restarting the program will create new data.", func(confirm bool) {
		if confirm {
			err := os.RemoveAll(owlcmsInstallDir)
			if err != nil {
				log.Printf("Failed to remove all data: %v\n", err)
				dialog.ShowError(fmt.Errorf("failed to remove all data: %w", err), fyne.CurrentApp().Driver().AllWindows()[0])
			} else {
				log.Println("All data removed successfully")
				dialog.ShowInformation("Success", "All data removed successfully", fyne.CurrentApp().Driver().AllWindows()[0])
				fyne.CurrentApp().Quit()
			}
		}
	}, fyne.CurrentApp().Driver().AllWindows()[0])
}

func removeJava() {
	javaDir := filepath.Join(owlcmsInstallDir, "java17")
	err := os.RemoveAll(javaDir)
	if err != nil {
		log.Printf("Failed to remove Java: %v\n", err)
		dialog.ShowError(fmt.Errorf("failed to remove Java: %w", err), fyne.CurrentApp().Driver().AllWindows()[0])
	} else {
		log.Println("Java removed successfully")
		dialog.ShowInformation("Success", "Java removed successfully", fyne.CurrentApp().Driver().AllWindows()[0])
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting owlcms-firmata Launcher")
	a := app.NewWithID("app.owlcmx.firmata-launcher")
	a.Settings().SetTheme(newMyTheme())
	w := a.NewWindow("owlcms-firmata Control Panel")
	w.Resize(fyne.NewSize(800, 400)) // Larger initial window size

	// Create stop button and status label
	stopButton = widget.NewButtonWithIcon("Stop", theme.CancelIcon(), nil)
	stopButton.Importance = widget.DangerImportance // This makes it red
	statusLabel = widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord // Allow status messages to wrap

	// Create containers
	downloadContainer = container.NewVBox()
	versionContainer = container.NewVBox()

	// Create URL hyperlink
	urlLink = widget.NewHyperlink("", nil)
	urlLink.Hide()

	stopContainer = container.NewVBox(stopButton, statusLabel, urlLink)

	// Initialize download titles
	updateTitle = widget.NewRichTextFromMarkdown("")                                             // Initialize as RichText for Markdown
	downloadButtonTitle = widget.NewHyperlink("Click here to install additional versions.", nil) // New title for download button
	downloadButtonTitle.OnTapped = func() {
		if !downloadsShown {
			ShowDownloadables()
		} else {
			HideDownloadables()
		}
	}
	singleOrMultiVersionLabel = widget.NewLabel("")

	// Configure stop button behavior
	stopButton.OnTapped = func() {
		log.Println("Stop button tapped")
		stopProcess(currentProcess, currentVersion, stopButton, downloadContainer, versionContainer, statusLabel, w)
	}
	stopButton.Hide()
	stopContainer.Hide()

	mainContent := container.NewVBox(
		stopContainer,
		versionContainer,
		downloadContainer, // Use downloadGroup here
	)
	statusLabel.SetText("Checking installation status...")
	statusLabel.Refresh()
	statusLabel.Show()
	stopContainer.Show()

	var javaAvailable bool
	go func() {
		javaLoc, err := javacheck.FindLocalJava()
		javaAvailable = err == nil && javaLoc != ""

		// Check for internet connection before anything else
		internetAvailable := CheckForInternet()
		if internetAvailable && !javaAvailable {
			// Check for Java before anything else
			if err := checkJava(statusLabel); err != nil {
				dialog.ShowError(fmt.Errorf("failed to fetch Java: %w", err), w)
			}
		}

		var releases []string
		if internetAvailable {
			releases, err = fetchReleases()
			if err == nil {
				allReleases = releases
			}
		} else {
			allReleases = []string{}
		}

		numVersions := len(getAllInstalledVersions())
		if numVersions == 0 && !internetAvailable {
			w.SetContent(mainContent)
			d := dialog.NewInformation("No Internet Connection", "You must be connected to the internet to fetch a version of the program.\nPlease connect and restart the program", w)
			d.Resize(fyne.NewSize(400, 200))
			d.SetDismissText("Exit")
			d.Show()
			d.SetOnClosed(func() {
				a.Driver().Quit()
			})
			return
		}

		// Initialize version list
		recomputeVersionList(w)

		// Create release dropdown for downloads
		releaseSelect, releaseDropdown := createReleaseDropdown(w)
		updateTitle.Hide()
		releaseDropdown.Hide() // Hide the dropdown initially

		if len(allReleases) > 0 {
			downloadContainer.Objects = []fyne.CanvasObject{
				updateTitle,
				singleOrMultiVersionLabel,
				downloadButtonTitle,
				releaseDropdown,
			}
		} else {
			downloadContainer.Objects = []fyne.CanvasObject{
				widget.NewLabel("You are not connected to the Internet. Available updates cannot be shown."),
			}
		}

		// Create menu items
		fileMenu := fyne.NewMenu("File",
			fyne.NewMenuItem("Remove All Versions", func() {
				removeAllVersions()
			}),
			fyne.NewMenuItem("Remove Java", func() {
				removeJava()
			}),
			fyne.NewMenuItem("Remove All Stored Data and Configurations", func() {
				uninstallAll()
			}),
			fyne.NewMenuItem("Open Installation Directory", func() {
				if err := openFileExplorer(owlcmsInstallDir); err != nil {
					dialog.ShowError(fmt.Errorf("failed to open installation directory: %w", err), w)
				}
			}),
		)
		killMenu := fyne.NewMenu("Processes",
			fyne.NewMenuItem("Kill Already Running Process", func() {
				if err := killLockingProcess(); err != nil {
					dialog.ShowError(fmt.Errorf("failed to kill already running process: %w", err), w)
				} else {
					dialog.ShowInformation("Success", "Successfully killed the already running process", w)
				}
			}),
		)
		helpMenu := fyne.NewMenu("Help",
			fyne.NewMenuItem("Documentation", func() {
				linkURL, _ := url.Parse("https://firmata.github.io/owlcms4-prerelease/#/LocalControlPanel")
				link := widget.NewHyperlink("Control Panel Documentation", linkURL)
				dialog.ShowCustom("Documentation", "Close", link, w)
			}),
			fyne.NewMenuItem("Check for Updates", func() {
				checkForUpdates(w)
			}),
			fyne.NewMenuItem("About", func() {
				dialog.ShowInformation("About", "owlcms-firmata Launcher version "+launcherVersion, w)
			}),
		)
		menu := fyne.NewMainMenu(fileMenu, killMenu, helpMenu)
		w.SetMainMenu(menu)
		mainContent.Resize(fyne.NewSize(800, 400))
		w.SetContent(mainContent)
		w.Resize(fyne.NewSize(800, 400))
		w.Canvas().Refresh(mainContent)

		populateReleaseSelect(releaseSelect) // Populate the dropdown with the releases
		updateTitle.Show()
		releaseDropdown.Hide()
		log.Printf("Fetched %d releases\n", len(releases))

		// If no version is installed, get the latest stable version
		if len(getAllInstalledVersions()) == 0 {
			for _, release := range allReleases {
				version := extractSemverTag(release)
				if !containsPreReleaseTag(version) {
					// Automatically download and install the latest stable version
					log.Printf("Downloading and installing latest stable version %s\n", version)
					downloadAndInstallVersion(version, w)
					break
				}
			}
		}

		// Check if a more recent version is available
		checkForNewerVersion()
		downloadContainer.Refresh()
		downloadContainer.Show()
		mainContent.Refresh()

		w.SetContent(mainContent)
		w.Canvas().Refresh(mainContent)

		w.SetCloseIntercept(func() {
			if currentProcess != nil {
				confirmDialog := dialog.NewConfirm(
					"Confirm Exit",
					"The server is running. This will stop the firmata server for all the users. Are you sure you want to exit?",
					func(confirm bool) {
						if !confirm {
							log.Println("Closing owlcms-firmata Launcher")
							stopProcess(currentProcess, currentVersion, stopButton, downloadContainer, versionContainer, statusLabel, w)
							w.Close()
						}
					},
					w,
				)
				confirmDialog.SetConfirmText("Don't Stop firmata")
				confirmDialog.SetDismissText("Stop firmata and Exit")
				confirmDialog.Show()
			} else {
				w.Close()
			}
		})

		log.Println("setup done.")
		statusLabel.Hide()
	}()

	// Set up channel to listen for interrupt signals BEFORE ShowAndRun
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var wg sync.WaitGroup

	// Goroutine to handle interrupt signal
	go func() {
		<-sigChan
		log.Println("Interrupt signal caught, stopping Java process...")
		wg.Add(1)
		go func() {
			defer wg.Done()
			stopProcess(currentProcess, currentVersion, stopButton, downloadContainer, versionContainer, statusLabel, w)
		}()
		wg.Wait()
		log.Println("Exiting Control Panel...")
		os.Exit(0)
	}()

	log.Println("Showing owlcms-firmata Launcher")
	w.ShowAndRun()
}

func HideDownloadables() {
	downloadsShown = false
	releaseDropdown.Hide()
	downloadContainer.Refresh()
}

func ShowDownloadables() {
	downloadsShown = true
	releaseDropdown.Show()
	downloadContainer.Refresh()
}

func downloadAndInstallVersion(version string, w fyne.Window) {
	var urlPrefix string
	if containsPreReleaseTag(version) {
		urlPrefix = "https://github.com/jflamy/owlcms-firmata/releases/download"
	} else {
		urlPrefix = "https://github.com/jflamy/owlcms-firmata/releases/download"
	}
	fileName := "owlcms-firmata.jar"
	zipURL := fmt.Sprintf("%s/%s/%s", urlPrefix, version, fileName)

	// Ensure the firmata directory exists
	owlcmsDir := owlcmsInstallDir
	if _, err := os.Stat(owlcmsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(owlcmsDir, 0755); err != nil {
			dialog.ShowError(fmt.Errorf("creating firmata directory: %w", err), w)
			return
		}
	}

	// Show progress dialog
	progressDialog := dialog.NewCustom(
		"Installing owlcms-firmata",
		"Please wait...",
		widget.NewLabel("Downloading and extracting files..."),
		w)
	progressDialog.Show()

	go func() {
		extractPath := filepath.Join(owlcmsDir, version)
		os.Mkdir(extractPath, 0755)
		extractPath = filepath.Join(extractPath, fileName)

		// Download the file using downloadUtils
		log.Printf("Starting download from URL: %s\n", zipURL)
		err := downloadUtils.DownloadArchive(zipURL, extractPath)
		if err != nil {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("download failed: %w", err), w)
			return
		}

		// Log when extraction is done
		log.Println("Extraction completed")

		// Hide progress dialog
		progressDialog.Hide()

		// Show success panel with installation details
		message := fmt.Sprintf(
			"Successfully installed owlcms-firmata version %s\n\n"+
				"Location: %s\n\n"+
				"The program files have been extracted to the above directory.",
			version, extractPath)

		dialog.ShowInformation("Installation Complete", message, w)
		HideDownloadables()

		// Recompute the version list
		recomputeVersionList(w)

		// Recompute the downloadTitle
		checkForNewerVersion()
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func checkForNewerVersion() {
	latestInstalled = findLatestInstalled()

	if latestInstalled != "" {
		latestInstalledVersion, err := semver.NewVersion(latestInstalled)
		if err == nil {
			log.Printf("Latest installed version: %s\n", latestInstalledVersion)
			latestStableString, stableErr := getMostRecentStableRelease()
			if stableErr == nil {
				stableVersion, _ := semver.NewVersion(latestStableString)
				// Compare installed version with latest stable
				if stableVersion.GreaterThan(latestInstalledVersion) {
					releaseURL := fmt.Sprintf("https://github.com/jflamy/owlcms-firmata/releases/tag/%s", latestStableString)
					updateTitle.ParseMarkdown(fmt.Sprintf("**A more recent stable version %s is available.** [Release Notes](%s)", latestStableString, releaseURL))
					updateTitle.Refresh()
					updateTitle.Show()
					return
				}
			}

			// If we get here, no newer version was found
			releaseURL := fmt.Sprintf("https://github.com/jflamy/owlcms-firmata/releases/tag/%s", latestInstalled)
			if containsPreReleaseTag(latestInstalled) {
				updateTitle.ParseMarkdown(fmt.Sprintf("**You are using pre-release %s** [Release Notes](%s)", latestInstalled, releaseURL))
			} else {
				updateTitle.ParseMarkdown(fmt.Sprintf("**You are using stable version %s** [Release Notes](%s)", latestInstalled, releaseURL))
			}
			updateTitle.Refresh()
			updateTitle.Show()
			downloadButtonTitle.Show()
			if releaseDropdown != nil {
				releaseDropdown.Hide()
			}
			if downloadContainer != nil {
				downloadContainer.Refresh()
			}
		}
	} else {
		updateTitle.ParseMarkdown("No version is installed.")
		updateTitle.Refresh()
		updateTitle.Show()
	}
}
