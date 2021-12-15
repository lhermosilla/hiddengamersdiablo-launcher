package config

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/google/uuid"
	"github.com/lhermosilla/hiddengamersdiablo-launcher/storage"
)

// Service is responsible for all things related to configuration.
type Service interface {
	// Read will read the configuration and return it.
	Read() (*storage.Config, error)

	// AddGame adds a new game to the game model.
	AddGame()

	// UpsertGame updates or creates a new game to the persistent store.
	UpsertGame(request UpdateGameRequest) error

	// DeleteGame will delete a game from the game model and the persistent store.
	DeleteGame(id string) error

	// PersistGameModel will persist the current game model to the persistent store.
	PersistGameModel() error

	// UpdateLaunchDelay will update the launch delay for  games in the persistent store.
	UpdateLaunchDelay(delay int) error

	// GetAvailableMods will fetch the game mode available to each D2 install.
	GetAvailableMods() (*GameMods, error)
}

type service struct {
	hiddengamersdiabloClient hiddengamersdiablo.Client
	store                    storage.Store
	gameModel                *GameModel
	mutex                    sync.Mutex
}

// Read will read the configuration and return it.
func (s *service) Read() (*storage.Config, error) {
	conf, err := s.store.Read()
	if err != nil {
		return nil, err
	}

	return conf, err
}

// AddGame adds a new game to the game model.
func (s *service) AddGame() {
	// Lock before we update the model preventing race conditions.
	s.mutex.Lock()

	// Unlock when we're done.
	defer s.mutex.Unlock()

	g := NewGame(nil)

	// Generate an ID for the new game.
	g.ID = uuid.New().String()

	// Default values.
	g.Instances = 1
	g.Flags = []string{"-w", "-skiptobnet"}
	g.HDVersion = ModVersionNone
	g.MaphackVersion = ModVersionNone

	s.gameModel.AddGame(g)
}

// UpdateGameRequest is the data used to update a game in the game model.
type UpdateGameRequest struct {
	ID             string   `json:"id"`
	Location       string   `json:"location"`
	Instances      int      `json:"instances"`
	OverrideBHCfg  bool     `json:"override_bh_cfg"`
	Flags          []string `json:"flags"`
	HDVersion      string   `json:"hd_version"`
	MaphackVersion string   `json:"maphack_version"`
}

// UpsertGame will upsert the game to the config.
func (s *service) UpsertGame(request UpdateGameRequest) error {
	// Lock before we update the model preventing race conditions.
	s.mutex.Lock()

	// Unlock when we're done.
	defer s.mutex.Unlock()

	// Updates game model with the new information.
	var updatedIndex int
	games := s.gameModel.Games()
	for i := 0; i < len(games); i++ {
		if games[i].ID == request.ID {
			updatedIndex = i
			games[i].Location = request.Location
			games[i].Instances = request.Instances
			games[i].OverrideBHCfg = request.OverrideBHCfg
			games[i].Flags = request.Flags
			games[i].HDVersion = request.HDVersion
			games[i].MaphackVersion = request.MaphackVersion
		}
	}

	// Notify the UI of the change.
	s.gameModel.updateGame(updatedIndex)

	return nil
}

// DeleteGame will delete the game from the config.
func (s *service) DeleteGame(id string) error {
	// Lock before we update the model preventing race conditions.
	s.mutex.Lock()

	// Unlock when we're done.
	defer s.mutex.Unlock()

	// Read the config in order to update it.
	conf, err := s.store.Read()
	if err != nil {
		return err
	}

	// Delete game from the config.
	for i := 0; i < len(conf.Games); i++ {
		if conf.Games[i].ID == id {
			// Remove the index from the game slice.
			conf.Games = append(conf.Games[:i], conf.Games[i+1:]...)
		}
	}

	// Write the new games slice to the config.
	err = s.store.Write(conf)
	if err != nil {
		return err
	}

	// Delete from the game model too.
	games := s.gameModel.Games()
	for i := 0; i < len(games); i++ {
		if games[i].ID == id {
			s.gameModel.removeGame(i)
		}
	}

	return nil
}

// PersistGameModel will persist the current game model to the persistent store.
func (s *service) PersistGameModel() error {
	conf, err := s.store.Read()
	if err != nil {
		return err
	}

	// Fetch the current game model.
	games := s.gameModel.Games()

	// Reset the games in the config.
	conf.Games = make([]storage.Game, 0)

	// Go through all games and populate a config slice.
	for i := 0; i < len(games); i++ {
		conf.Games = append(conf.Games, storage.Game{
			ID:             games[i].ID,
			Location:       games[i].Location,
			Instances:      games[i].Instances,
			OverrideBHCfg:  games[i].OverrideBHCfg,
			Flags:          games[i].Flags,
			HDVersion:      games[i].HDVersion,
			MaphackVersion: games[i].MaphackVersion,
		})
	}

	err = s.store.Write(conf)
	if err != nil {
		return err
	}

	return nil
}

// UpdateLaunchDelay will update the Diablo launch delay in the store.
func (s *service) UpdateLaunchDelay(delay int) error {
	conf, err := s.store.Read()
	if err != nil {
		return err
	}

	// Update launch delay.
	conf.LaunchDelay = delay

	err = s.store.Write(conf)
	if err != nil {
		return err
	}

	return nil
}

// GetAvailableMods will get available mods from the Slashdiablo API.
func (s *service) GetAvailableMods() (*GameMods, error) {
	contents, err := s.hiddengamersdiabloClient.GetAvailableMods()
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(contents)
	if err != nil {
		return nil, err
	}

	var gameMods GameMods
	if err := json.Unmarshal(bytes, &gameMods); err != nil {
		return nil, err
	}

	return &gameMods, nil
}

// NewService returns a service with all the dependencies.
func NewService(
	hiddengamersdiabloClient hiddengamersdiablo.Client,
	store storage.Store,
	gameModel *GameModel,
) Service {
	return &service{
		hiddengamersdiabloClient: hiddengamersdiabloClient,
		store:                    store,
		gameModel:                gameModel,
	}
}
