package d2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lhermosilla/hiddengamersdiablo-launcher/clients/hiddengamersdiablo"
	"github.com/lhermosilla/hiddengamersdiablo-launcher/config"
	"github.com/lhermosilla/hiddengamersdiablo-launcher/storage"
	"github.com/nokka/slashdiablo-launcher/log"
)

// Service is responsible for all things related to the Slashdiablo ladder.
type Service interface {
	// Exec is responsible for executing the Diablo II game.
	Exec() error

	// ValidateGameVersions will make sure the game is up to date with expected patch.
	ValidateGameVersions() (bool, error)

	// Patch will patch Diablo II to the correct version.
	Patch(done chan bool) (<-chan float32, <-chan PatchState)

	// ApplyDEP will apply Windows specific fix for DEP.
	ApplyDEP(path string) error

	// SetLaunchDelay is responsible for setting the delay between each game launch.
	SetLaunchDelay(delay int) error
}

// Service is responsible for all things related to Diablo II.
type service struct {
	hiddengamersdiabloClient hiddengamersdiablo.Client
	configService            config.Service
	logger                   log.Logger
	gameStates               chan execState
	availableMods            *config.GameMods
	runningGames             []game
	mux                      sync.Mutex
	patchFileModel           *FileModel
}

type game struct {
	PID    int
	GameID string
}

type execState struct {
	pid *int
	err error
}

// defaultLaunchDelay is used if a launch delay hasn't been set by a user.
const defaultLaunchDelay = 1000

// Exec will exec Diablo 2 installs.
func (s *service) Exec() error {
	conf, err := s.configService.Read()
	if err != nil {
		return err
	}

	// Mutate the number of instances to launch to take into
	// account the number of games already running.
	s.mutateInstancesToLaunch(conf.Games)

	var delayMS int
	if conf.LaunchDelay == 0 {
		delayMS = defaultLaunchDelay
	} else {
		delayMS = conf.LaunchDelay
	}

	for k, g := range conf.Games {
		if g.Instances > 0 {
			for i := 0; i < g.Instances; i++ {
				// Check if it's the first run, if so don't delay the launch.
				firstRun := (k == 0 && i == 0)

				if !firstRun {
					time.Sleep(time.Duration(delayMS) * time.Millisecond)
				}

				// The third argument is a channel, listened on by listenForGameStates().
				pid, err := launch(g.Location, g.Flags, s.gameStates)
				if err != nil {
					return err
				}

				// Add the started game to our slice of games.
				s.runningGames = append(s.runningGames, game{PID: *pid, GameID: g.ID})
			}
		}
	}

	return nil
}

func (s *service) getAvailableMods() (*config.GameMods, error) {
	// Return cached available mods.
	if s.availableMods != nil {
		return s.availableMods, nil
	}

	// No cached mods exist, fetch remote mods.
	contents, err := s.hiddengamersdiabloClient.GetAvailableMods()
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(contents)
	if err != nil {
		return nil, err
	}

	var gameMods config.GameMods
	if err := json.Unmarshal(bytes, &gameMods); err != nil {
		return nil, err
	}

	// Set cache.
	s.availableMods = &gameMods

	return s.availableMods, nil
}

// ValidateGameVersions will check if the games are up to date.
func (s *service) ValidateGameVersions() (bool, error) {
	conf, err := s.configService.Read()
	if err != nil {
		return false, err
	}

	// Get current slash patch and compare.
	version113cManifest, err := s.getManifest("1.13c/manifest.json")
	if err != nil {
		return false, err
	}

	// Get current slash patch and compare.
	slashManifest, err := s.getManifest("current/manifest.json")
	if err != nil {
		return false, err
	}

	mods, err := s.getAvailableMods()
	if err != nil {
		return false, err
	}

	upToDate := true

	if len(conf.Games) > 0 {
		for _, game := range conf.Games {
			valid, err := validate113cVersion(game.Location)
			if err != nil {
				return false, err
			}

			// Game wasn't 1.13c, needs to be updated.
			if !valid {
				upToDate = false
				// Get files that aren't up to date and add them to the file model.
				version113cFiles, _, err := s.getFilesToPatch(version113cManifest.Files, game.Location, nil)
				if err != nil {
					return false, err
				}

				s.addFilesToModel(version113cFiles)
			}

			// Check if the current game install is up to date with the slash patch.
			slashFiles, _, err := s.getFilesToPatch(slashManifest.Files, game.Location, nil)
			if err != nil {
				return false, err
			}

			// Slash patch isn't up to date.
			if len(slashFiles) > 0 {
				s.addFilesToModel(slashFiles)
				upToDate = false
			}

			validMaphack, err := s.validateMaphackVersion(&game, mods.Maphack)
			if err != nil {
				return false, err
			}

			// Maphack version wasn't valid, we need to update.
			if !validMaphack {
				upToDate = false
			}

			validHD, err := s.validateHDVersion(&game, mods.HD)
			if err != nil {
				return false, err
			}

			// HD version wasn't valid, we need to update.
			if !validHD {
				upToDate = false
			}
		}
	}

	// Games are both 1.13c and up to date with Slash patch, maphack and HD.
	return upToDate, nil
}

func (s *service) resetPatch(path string, files []PatchFile, filesToIgnore []string) error {
	// Check how many files aren't up to date.
	missmatchedFiles, _, err := s.getFilesToPatch(files, path, filesToIgnore)
	if err != nil {
		return err
	}

	// If the number of missmatched files to patch aren't all of them, then we have
	// some of them left that needs to be removed.
	if len(missmatchedFiles) != len(files) {
		for _, file := range files {
			filePath := localizePath(fmt.Sprintf("%s/%s", path, file.Name))

			// Check if the file exists, on disk, if it does, remove it.
			_, err := os.Stat(filePath)
			if err != nil {
				// File didn't exist on disk, continue to next.
				if os.IsNotExist(err) {
					continue
				}
				// Unknown error.
				return err
			}

			// Make sure we don't remove the ignored files.
			var ignore bool

			for _, ignored := range filesToIgnore {
				if file.Name == ignored {
					ignore = true
					break
				}
			}

			if !ignore {
				// File that shouldn't be on disk exists, remove it.
				err = os.Remove(filePath)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *service) resetHDPatch(game storage.Game) error {
	mods, err := s.getAvailableMods()
	if err != nil {
		return err
	}

	// Go over available HD mods and reset them if they are installed.
	for _, m := range mods.HD {
		// Desired version, don't reset it.
		if game.HDVersion == m {
			continue
		}

		HDManifest, err := s.getManifest(fmt.Sprintf("hd_%s/manifest.json", m))
		if err != nil {
			return err
		}

		installed, err := isModInstalled(game.Location, ModHDIdentifier, HDManifest)
		if err != nil {
			return err
		}

		if installed {
			err := s.resetPatch(game.Location, HDManifest.Files, nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *service) resetMaphackPatch(game storage.Game, filesToIgnore []string) error {
	mods, err := s.getAvailableMods()
	if err != nil {
		return err
	}

	// Go over available maphack versions and reset them if they are installed.
	for _, m := range mods.Maphack {
		// Desired version, don't reset it.
		if game.MaphackVersion == m {
			continue
		}

		maphackManifest, err := s.getManifest(fmt.Sprintf("maphack_%s/manifest.json", m))
		if err != nil {
			return err
		}

		installed, err := isModInstalled(game.Location, ModMaphackIdentifier, maphackManifest)
		if err != nil {
			return err
		}

		if installed {
			err := s.resetPatch(game.Location, maphackManifest.Files, filesToIgnore)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Patch will check for updates and if found, patch the game, both D2 and HD version.
func (s *service) Patch(done chan bool) (<-chan float32, <-chan PatchState) {
	progress := make(chan float32)
	state := make(chan PatchState)

	go func() {
		conf, err := s.configService.Read()
		if err != nil {
			state <- PatchState{Error: err}
			return
		}

		// Map of HD manifests, so we don't have to download them twice.
		var hdManifests = make(map[string]*Manifest, 0)

		// Map of maphack manifests, so we don't have to download them twice.
		var maphackManifests = make(map[string]*Manifest, 0)

		for _, game := range conf.Games {
			// If the user has chosen to override the maphack config with their own,
			// we need to make sure the config is being ignored from the patch, and also
			// when reseting the maphack patch.
			var ignoredMaphackFiles []string

			if game.OverrideBHCfg {
				ignoredMaphackFiles = append(ignoredMaphackFiles, "BH.cfg")
			}

			// Reset the maphack versions, to avoid rogue files and duplicates.
			err := s.resetMaphackPatch(game, ignoredMaphackFiles)
			if err != nil {
				state <- PatchState{Error: err}
				return
			}

			// Reset the HD versions, to avoid rogue files and duplicates.
			err = s.resetHDPatch(game)
			if err != nil {
				state <- PatchState{Error: err}
				return
			}

			// The install has been reset, let's validate the 1.13c version and apply missing files.
			if err := s.apply113c(game.Location, state, progress); err != nil {
				state <- PatchState{Error: err}
				return
			}

			// Apply the Slashdiablo specific patch.
			err = s.applySlashPatch(game.Location, state, progress)
			if err != nil {
				state <- PatchState{Error: err}
				return
			}

			// Maphack version was set on the game, download it.
			if game.MaphackVersion != config.ModVersionNone {
				mm, ok := maphackManifests[game.MaphackVersion]
				if !ok {
					maphackManifest, err := s.getManifest(fmt.Sprintf("maphack_%s/manifest.json", game.MaphackVersion))
					if err != nil {
						state <- PatchState{Error: err}
						return
					}

					maphackManifests[game.MaphackVersion] = maphackManifest
					mm = maphackManifest
				}

				// Just to be safe and avoid a panic.
				if maphackManifests[game.MaphackVersion] == nil {
					state <- PatchState{Error: errors.New("no hay manifest del mh")}
					return
				}

				err = s.applyMaphack(game.Location, game.MaphackVersion, state, progress, mm.Files, ignoredMaphackFiles)
				if err != nil {
					state <- PatchState{Error: err}
					return
				}
			}

			// HD version was set on the game, download it.
			if game.HDVersion != config.ModVersionNone {
				hdm, ok := hdManifests[game.HDVersion]
				if !ok {
					hdManifest, err := s.getManifest(fmt.Sprintf("hd_%s/manifest.json", game.HDVersion))
					if err != nil {
						state <- PatchState{Error: err}
						return
					}

					hdManifests[game.HDVersion] = hdManifest
					hdm = hdManifest
				}

				// Just to be safe and avoid a panic.
				if hdManifests[game.HDVersion] == nil {
					state <- PatchState{Error: errors.New("no hay manifest del hd")}
					return
				}

				err = s.applyHDMod(game.Location, game.HDVersion, state, progress, hdm.Files)
				if err != nil {
					state <- PatchState{Error: err}
					return
				}
			}

			// Finally set os specific configurations, such as compatibility mode.
			err = configureForOS(game.Location)
			if err != nil {
				state <- PatchState{Error: err}
				return
			}
		}

		done <- true
	}()

	return progress, state
}

// ApplyDEP will run  data execution prevention (DEP) on the Game.exe in the path.
func (s *service) ApplyDEP(path string) error {
	// Run OS specific fix.
	return applyDEP(path)
}

// SetLaunchDelay will set the given delay in milliseconds between each Diablo II launch.
func (s *service) SetLaunchDelay(delay int) error {
	// Set launch delay in the config.
	err := s.configService.UpdateLaunchDelay(delay)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) mutateInstancesToLaunch(games []storage.Game) {
	for i := 0; i < len(games); i++ {
		var runningCount int
		for _, running := range s.runningGames {
			if games[i].ID == running.GameID {
				runningCount++
			}
		}

		// If any games of this id is running already, subtract the number
		// and mutate the game so the next time we launch, we launch the correct number.
		games[i].Instances = games[i].Instances - runningCount
	}
}

func (s *service) listenForGameStates() {
	for {
		select {
		case state := <-s.gameStates:
			// Something went wrong while execing, log error.
			if state.err != nil {
				s.logger.Error(fmt.Errorf("Diablo II exec con codigo: %s", state.err))
			}

			s.mux.Lock()

			// Game exited, remove it from the slice based on pid.
			for index, g := range s.runningGames {
				if state.pid != nil && g.PID == *state.pid {
					s.runningGames = append(s.runningGames[:index], s.runningGames[index+1:]...)
				}
			}

			s.mux.Unlock()
		}
	}
}

func (s *service) validateMaphackVersion(game *storage.Game, versions []string) (bool, error) {
	isValid := true

	for _, v := range versions {
		manifest, err := s.getManifest(fmt.Sprintf("maphack_%s/manifest.json", v))
		if err != nil {
			return false, err
		}

		// If the user has chosen to override the maphack config with their own,
		// we need to make sure the config is being ignored from the patch.
		var ignoredMaphackFiles []string

		if game.OverrideBHCfg {
			ignoredMaphackFiles = append(ignoredMaphackFiles, "BH.cfg")
		}
		// This particular maphack version should be installed.
		if game.MaphackVersion == v {
			// Check how many files aren't up to date with maphack.
			missingMaphackFiles, _, err := s.getFilesToPatch(manifest.Files, game.Location, ignoredMaphackFiles)
			if err != nil {
				return false, err
			}

			// Maphack patch isn't up to date.
			if len(missingMaphackFiles) > 0 {
				s.addFilesToModel(missingMaphackFiles)
				isValid = false
			}
		} else {
			installed, err := isModInstalled(game.Location, ModMaphackIdentifier, manifest)
			if err != nil {
				return false, err
			}

			// Maphack wasn't supposed to be installed, but it is, we need to update.
			if installed {
				// Before we return, we need to add these to the patch actions, since they will be removed.
				err := s.addPatchFilesToBeDeleted(game.Location, manifest.Files)
				if err != nil {
					return false, err
				}

				isValid = false
			}
		}
	}

	return isValid, nil
}

func (s *service) validateHDVersion(game *storage.Game, versions []string) (bool, error) {
	isValid := true

	for _, v := range versions {
		manifest, err := s.getManifest(fmt.Sprintf("hd_%s/manifest.json", v))
		if err != nil {
			return false, err
		}

		// This particular HD version should be installed.
		if game.HDVersion == v {
			// Check if the current game install is up to date with the HD patch.
			missingFiles, _, err := s.getFilesToPatch(manifest.Files, game.Location, nil)
			if err != nil {
				return false, err
			}

			// HD mod isn't up to date.
			if len(missingFiles) > 0 {
				s.addFilesToModel(missingFiles)
				isValid = false
			}
		} else {
			installed, err := isModInstalled(game.Location, ModHDIdentifier, manifest)
			if err != nil {
				return false, err
			}

			// HD wasn't supposed to be installed, but it is, we need to update.
			if installed {
				// Before we return, we need to add these to the patch actions, since they will be removed.
				err := s.addPatchFilesToBeDeleted(game.Location, manifest.Files)
				if err != nil {
					return false, err
				}

				isValid = false
			}
		}
	}

	return isValid, nil
}

func (s *service) apply113c(path string, state chan PatchState, progress chan float32) error {
	state <- PatchState{Message: "Comprobando version del juego..."}

	// Download manifest from patch repository.
	manifest, err := s.getManifest("1.13c/manifest.json")
	if err != nil {
		return err
	}

	// Figure out which files to patch.
	patchFiles, patchLength, err := s.getFilesToPatch(manifest.Files, path, nil)
	if err != nil {
		return err
	}

	if len(patchFiles) > 0 {
		state <- PatchState{Message: fmt.Sprintf("Actualizando %s a 1.13c", path)}
		if err := s.doPatch(patchFiles, patchLength, "1.13c", path, progress); err != nil {
			patchErr := err
			// Make sure we clean up the failed patch.
			if err := s.cleanUpFailedPatch(path); err != nil {
				return fmt.Errorf("Error de limpieza: %s : %s", patchErr, err)
			}

			return err
		}
	}

	return nil
}

func (s *service) applySlashPatch(path string, state chan PatchState, progress chan float32) error {
	state <- PatchState{Message: "Comprobando parche de HiddenGamers Diablo..."}

	// Download manifest from patch repository.
	manifest, err := s.getManifest("current/manifest.json")
	if err != nil {
		return err
	}

	// Figure out which files to patch.
	patchFiles, patchLength, err := s.getFilesToPatch(manifest.Files, path, nil)
	if err != nil {
		return err
	}

	if len(patchFiles) > 0 {
		state <- PatchState{Message: fmt.Sprintf("Actualizando %s al parche actual de HiddenGamers Diablo", path)}

		if err = s.doPatch(patchFiles, patchLength, "current", path, progress); err != nil {
			patchErr := err
			// Make sure we clean up the failed patch.
			if err := s.cleanUpFailedPatch(path); err != nil {
				return fmt.Errorf("Error de limpieza: %s : %s", patchErr, err)
			}

			return err
		}
	}

	return nil
}

func (s *service) applyMaphack(path string, version string, state chan PatchState, progress chan float32, manifestFiles []PatchFile, ignoredFiles []string) error {
	state <- PatchState{Message: "Comprobando version del Maphack.."}

	// Figure out which files to patch.
	patchFiles, patchLength, err := s.getFilesToPatch(manifestFiles, path, ignoredFiles)
	if err != nil {
		return err
	}

	if len(patchFiles) > 0 {
		state <- PatchState{Message: fmt.Sprintf("Actualizando %s a la version del maphack %s", path, version)}

		remoteDir := fmt.Sprintf("maphack_%s", version)

		if err = s.doPatch(patchFiles, patchLength, remoteDir, path, progress); err != nil {
			patchErr := err
			// Make sure we clean up the failed patch.
			if err := s.cleanUpFailedPatch(path); err != nil {
				return fmt.Errorf("Error de limpieza: %s : %s", patchErr, err)
			}

			return err
		}
	}

	return nil
}

func (s *service) applyHDMod(path string, version string, state chan PatchState, progress chan float32, manifestFiles []PatchFile) error {
	// Update UI.
	state <- PatchState{Message: "Comprobando version del Mod HD..."}

	// Figure out which files to patch.
	patchFiles, patchLength, err := s.getFilesToPatch(manifestFiles, path, nil)
	if err != nil {
		return err
	}

	if len(patchFiles) > 0 {
		// Update UI.
		state <- PatchState{Message: fmt.Sprintf("Actualizando %s la version %s del mod HD", path, version)}

		remoteDir := fmt.Sprintf("hd_%s", version)

		if err = s.doPatch(patchFiles, patchLength, remoteDir, path, progress); err != nil {
			patchErr := err
			// Make sure we clean up the failed patch.
			if err := s.cleanUpFailedPatch(path); err != nil {
				return fmt.Errorf("Error de limpieza: %s : %s", patchErr, err)
			}

			return err
		}
	}

	return nil
}

func (s *service) doPatch(patchFiles []PatchAction, patchLength int64, remoteDir string, path string, progress chan float32) error {
	// Reset progress.
	progress <- 0.00

	// Create a write counter that will get bytes written per cycle, pass the
	// progress channel to report the number of bytes written.
	counter := &WriteCounter{
		Total:    float32(patchLength),
		progress: progress,
	}

	// Store the downloaded .tmp suffixed files.
	var tmpFiles []string

	// Patch the files.
	for _, action := range patchFiles {
		// Create the file, but give it a tmp file extension, this means we won't overwrite a
		// file until it's downloaded, but we'll remove the tmp extension once downloaded.
		tmpPath := localizePath(fmt.Sprintf("%s/%s.tmp", path, action.File.Name))

		switch action.Action {
		case ActionDownload:
			err := s.downloadFile(action.File.Name, remoteDir, tmpPath, counter)
			if err != nil {
				return err
			}
			tmpFiles = append(tmpFiles, tmpPath)
		case ActionDelete:
			err := s.deleteFile(action.File.Name, path)
			if err != nil {
				return err
			}
		}
	}

	// All the files were successfully downloaded, remove the .tmp suffix
	// to complete the patch entirely.
	for _, tmpFile := range tmpFiles {
		err := os.Rename(tmpFile, tmpFile[:len(tmpFile)-4])
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) downloadFile(fileName string, remoteDir string, path string, counter *WriteCounter) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}

	defer out.Close()

	f := fmt.Sprintf("%s/%s", remoteDir, fileName)
	contents, err := s.hiddengamersdiabloClient.GetFile(f)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, io.TeeReader(contents, counter))
	if err != nil {
		return err
	}

	return nil
}

func fileExistsOnDisk(fileName string, path string) (bool, error) {
	filePath := localizePath(fmt.Sprintf("%s/%s", path, fileName))

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		// Any other error, return it.
		return false, err
	}

	return true, nil
}

func (s *service) deleteFile(fileName string, path string) error {
	filePath := localizePath(fmt.Sprintf("%s/%s", path, fileName))

	// Check that the file exists.
	_, err := os.Stat(filePath)
	if err != nil {
		// File didn't exist on disk, just return nil.
		if os.IsNotExist(err) {
			return nil
		}
		// Any other error, return it.
		return err
	}

	// File exists on disk, so let's remove it.
	err = os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) cleanUpFailedPatch(dir string) error {
	files, err := ioutil.ReadDir(localizePath(dir))
	if err != nil {
		return err
	}

	for _, f := range files {
		fileName := f.Name()
		if strings.Contains(fileName, ".tmp") {
			err := os.Remove(localizePath(fmt.Sprintf("%s/%s", dir, fileName)))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *service) getFilesToPatch(files []PatchFile, d2path string, filesToIgnore []string) ([]PatchAction, int64, error) {
	shouldPatch := make([]PatchAction, 0)
	var totalContentLength int64

	for _, file := range files {
		f := file

		// Full path on disk to the patch file.
		localPath := localizePath(fmt.Sprintf("%s/%s", d2path, f.Name))

		// Check if the file should be ignored or not.
		if filesToIgnore != nil && len(filesToIgnore) > 0 {
			var ignore bool
			for _, ignored := range filesToIgnore {
				// If the current file should be ignored, just skip it.
				if f.Name == ignored {
					ignore = true
					break
				}
			}

			// File should be ignored, continue with the next.
			if ignore {
				continue
			}
		}
		// Check if file has been deprecated.
		if f.Deprecated {
			exists, err := fileExistsOnDisk(f.Name, d2path)
			if err != nil {
				return nil, 0, err
			}

			// If it still exists locally, queue it to be removed.
			if exists {
				// Get the checksum from the patch file on disk.
				hashed, err := hashCRC32(localPath, polynomial)
				if err != nil {
					return nil, 0, err
				}
				shouldPatch = append(shouldPatch, PatchAction{
					File:     f,
					Action:   ActionDelete,
					LocalCRC: hashed,
					D2Path:   d2path,
				})
			}

			continue
		}

		// Get the checksum from the patch file on disk.
		hashed, err := hashCRC32(localPath, polynomial)

		if err != nil {
			// If the file doesn't exist on disk, we need to patch it.
			if err == ErrCRCFileNotFound {
				shouldPatch = append(shouldPatch, PatchAction{
					File:     f,
					Action:   ActionDownload,
					LocalCRC: hashed,
					D2Path:   d2path,
				})
				totalContentLength += f.ContentLength
				continue
			}

			// Any other error, just return it.
			return nil, 0, err
		}

		// If the file is set to ignore the CRC, this means we don't want
		// to patch it, even if the content has been changed, as long as it
		// exists on disk we're good.
		if f.IgnoreCRC {
			continue
		}

		// File checksum differs from local copy, we need to get a new one.
		if hashed != f.CRC {
			shouldPatch = append(shouldPatch, PatchAction{
				File:     f,
				Action:   ActionDownload,
				LocalCRC: hashed,
				D2Path:   d2path,
			})
			totalContentLength += f.ContentLength
		}
	}

	return shouldPatch, totalContentLength, nil
}

func (s *service) getManifest(path string) (*Manifest, error) {
	contents, err := s.hiddengamersdiabloClient.GetFile(path)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(contents)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (s *service) addPatchFilesToBeDeleted(d2path string, files []PatchFile) error {
	var actions = make([]PatchAction, len(files))

	for i, file := range files {
		// Full path on disk to the patch file.
		localPath := localizePath(fmt.Sprintf("%s/%s", d2path, file.Name))

		hashed, err := hashCRC32(localPath, polynomial)
		if err != nil {
			return err
		}

		actions[i] = PatchAction{
			Action:   ActionDelete,
			File:     file,
			LocalCRC: hashed,
			D2Path:   d2path,
		}
	}

	s.addFilesToModel(actions)

	return nil
}

func (s *service) addFilesToModel(patchActions []PatchAction) {
	for _, action := range patchActions {
		f := NewFile(nil)
		f.Name = action.File.Name
		f.D2Path = action.D2Path
		f.RemoteCRC = action.File.CRC
		f.LocalCRC = action.LocalCRC
		f.FileAction = string(action.Action)
		s.patchFileModel.AddFile(f)
	}
}

// PatchState represents the state given on every patch cycle.
type PatchState struct {
	Message string
	Error   error
}

// Manifest represents the current patch.
type Manifest struct {
	Files []PatchFile `json:"files"`
}

// PatchFile represents a file that should be patched.
type PatchFile struct {
	Name          string    `json:"name"`
	CRC           string    `json:"crc"`
	LastModified  time.Time `json:"last_modified"`
	ContentLength int64     `json:"content_length"`
	IgnoreCRC     bool      `json:"ignore_crc"`
	Deprecated    bool      `json:"deprecated"`
}

// Action is an action performed while patching.
type Action string

// Allowed actions.
const (
	ActionDownload Action = "descargar"
	ActionDelete   Action = "eliminar"
)

// PatchAction is performed while patching.
type PatchAction struct {
	Action   Action
	File     PatchFile
	D2Path   string
	LocalCRC string
}

// NewService returns a service with all the dependencies.
func NewService(
	hiddengamersdiabloClient hiddengamersdiablo.Client,
	configuration config.Service,
	logger log.Logger,
	patchFileModel *FileModel,
) Service {
	s := &service{
		hiddengamersdiabloClient: hiddengamersdiabloClient,
		configService:            configuration,
		logger:                   logger,
		gameStates:               make(chan execState, 4),
		patchFileModel:           patchFileModel,
	}

	// Setup game listener once, will stay alive for the duration
	// of the service's life cycle.
	go s.listenForGameStates()

	return s
}
