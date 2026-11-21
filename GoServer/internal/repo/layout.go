package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"broadcast-tool/models"

	"github.com/google/uuid"
)

const storageSchema = 2
const archiveName = "songs-without-audio.json"
const journalName = ".storage-transaction.json"

type songLists struct {
	DormSongs      []models.Song `json:"dormSongs"`
	BroadcastSongs []models.Song `json:"broadcastSongs"`
}

type storedWeek struct {
	songLists
	Files map[string]models.WeekFile `json:"files,omitempty"`
}

type storageHeader struct {
	SchemaVersion int             `json:"storageVersion"`
	Settings      models.Settings `json:"settings"`
	NextSongID    int64           `json:"nextSongId"`
	Weeks         []string        `json:"weekDirectories"`
}

type FileMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type fileTransaction struct {
	Writes map[string]json.RawMessage `json:"writes"`
	Moves  []FileMove                 `json:"moves,omitempty"`
	Remove []string                   `json:"remove,omitempty"`
}

func weekForDate(date string) string {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return ""
	}
	year, week := t.ISOWeek()
	return fmt.Sprintf("%04d年第%d周", year, week)
}

func weekStart(name string) (time.Time, bool) {
	var year, week int
	if _, err := fmt.Sscanf(name, "%d年第%d周", &year, &week); err != nil ||
		year < 1970 || year > 9999 || week < 1 || week > 53 || name != fmt.Sprintf("%04d年第%d周", year, week) {
		return time.Time{}, false
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	start := jan4.AddDate(0, 0, -((int(jan4.Weekday())+6)%7)+7*(week-1))
	return start, weekForDate(start.Format(time.DateOnly)) == name
}

func weekMetadata(name string) string {
	return filepath.Join("MusicFiles", name, "songs.json")
}

func readJSON(root *os.Root, name string, dst any) error {
	data, err := root.ReadFile(name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func atomicFile(root *os.Root, name string, data []byte) error {
	if err := root.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	temp := name + "." + uuid.NewString() + ".tmp"
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temp)
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if err := errors.Join(err, f.Close()); err != nil {
		return err
	}
	return root.Rename(temp, name)
}

// Replay applies operations that were committed to the journal. The process
// lock protects the interval between writing the journal and clearing it.
func replay(root *os.Root, txn fileTransaction) error {
	for _, move := range txn.Moves {
		if err := root.MkdirAll(filepath.Dir(move.To), 0700); err != nil {
			return err
		}
		if err := root.Rename(move.From, move.To); err != nil {
			if _, moved := root.Lstat(move.To); !errors.Is(err, fs.ErrNotExist) || moved != nil {
				return fmt.Errorf("move %s to %s: %w", move.From, move.To, err)
			}
		}
	}
	for name, data := range txn.Writes {
		if name == "data.json" {
			continue
		}
		if err := atomicFile(root, name, data); err != nil {
			return err
		}
	}
	for _, name := range txn.Remove {
		if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := atomicFile(root, "data.json", txn.Writes["data.json"]); err != nil {
		return err
	}
	return root.Remove(journalName)
}

func loadStorage(root *os.Root, header storageHeader) (Data, error) {
	if header.SchemaVersion != storageSchema {
		return Data{}, fmt.Errorf("unsupported storage version %d", header.SchemaVersion)
	}
	var archive songLists
	if err := readJSON(root, archiveName, &archive); err != nil {
		return Data{}, err
	}
	d := Data{
		Settings: header.Settings, NextSongID: header.NextSongID,
		DormSongs: archive.DormSongs, BroadcastSongs: archive.BroadcastSongs,
		Weeks: make(map[string]map[string]models.WeekFile),
	}
	seen := make(map[string]bool)
	for _, name := range header.Weeks {
		if _, valid := weekStart(name); !valid || seen[name] {
			return Data{}, fmt.Errorf("invalid or duplicate week %q", name)
		}
		seen[name] = true
		var week storedWeek
		if err := readJSON(root, weekMetadata(name), &week); err != nil {
			return Data{}, err
		}
		d.DormSongs = append(d.DormSongs, week.DormSongs...)
		d.BroadcastSongs = append(d.BroadcastSongs, week.BroadcastSongs...)
		if week.Files != nil {
			d.Weeks[name] = week.Files
		}
	}
	sortPersistedSongs(d.DormSongs)
	sortPersistedSongs(d.BroadcastSongs)
	return d, nil
}

func sortPersistedSongs(songs []models.Song) {
	slices.SortStableFunc(songs, func(a, b models.Song) int {
		return strings.Compare(a.Date, b.Date)
	})
}

func managedAudio(path string) (string, bool) {
	if path == "" || strings.ContainsAny(path, ":") {
		return "", false
	}
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		parts = []string{"_staging", path}
	}
	if len(parts) != 2 || parts[0] == "" || filepath.Base(parts[1]) != parts[1] ||
		parts[1] == "" || parts[1] == "." || parts[1] == ".." || !filepath.IsLocal(filepath.Join(parts...)) ||
		!strings.EqualFold(filepath.Ext(parts[1]), ".mp3") {
		return "", false
	}
	if _, valid := weekStart(parts[0]); !valid && parts[0] != "_staging" {
		return "", false
	}
	return filepath.Join("MusicFiles", parts[0], parts[1]), true
}

func includeSongID(d *Data, songs []models.Song) {
	for _, song := range songs {
		if song.ID >= d.NextSongID {
			d.NextSongID = song.ID + 1
		}
	}
}

func (r *Repository) persist(d Data, moves []FileMove, additional map[string]json.RawMessage) error {
	root, err := os.OpenRoot(r.appDataDir)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Lstat(journalName); err == nil {
		return errors.New("pending storage transaction, reopen BCTools to recover")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	pendingFiles := make(map[string]bool)
	for _, move := range moves {
		pendingFiles[move.To] = true
	}
	weeks := make(map[string]*storedWeek)
	getWeek := func(name string) *storedWeek {
		if weeks[name] == nil {
			weeks[name] = &storedWeek{}
		}
		return weeks[name]
	}
	for name, index := range d.Weeks {
		if _, valid := weekStart(name); !valid {
			return fmt.Errorf("invalid week %q", name)
		}
		getWeek(name).Files = index
	}
	var archive songLists
	for _, list := range []struct {
		songs     []models.Song
		broadcast bool
	}{{d.DormSongs, false}, {d.BroadcastSongs, true}} {
		for _, song := range list.songs {
			song.Weekday = ""
			dest := &archive
			if name, ok := managedAudio(song.FilePath); ok {
				if info, err := root.Stat(name); ((err == nil && info.Mode().IsRegular()) || pendingFiles[name]) && weekForDate(song.Date) != "" {
					dest = &getWeek(weekForDate(song.Date)).songLists
				}
			}
			if list.broadcast {
				dest.BroadcastSongs = append(dest.BroadcastSongs, song)
			} else {
				dest.DormSongs = append(dest.DormSongs, song)
			}
		}
	}

	includeSongID(&d, d.DormSongs)
	includeSongID(&d, d.BroadcastSongs)
	if d.NextSongID < 1 {
		d.NextSongID = 1
	}
	names := make([]string, 0, len(weeks))
	for name := range weeks {
		names = append(names, name)
	}
	slices.Sort(names)
	header := storageHeader{storageSchema, d.Settings, d.NextSongID, names}
	txn := fileTransaction{Writes: make(map[string]json.RawMessage), Moves: moves}
	set := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		if current, err := root.ReadFile(name); name == "data.json" || err != nil || !bytes.Equal(current, b) {
			txn.Writes[name] = b
		}
		return nil
	}
	if err := set(archiveName, archive); err != nil {
		return err
	}
	for _, name := range names {
		if err := set(weekMetadata(name), weeks[name]); err != nil {
			return err
		}
	}
	if err := set("data.json", header); err != nil {
		return err
	}
	var previous storageHeader
	if err := readJSON(root, "data.json", &previous); err == nil {
		for _, name := range previous.Weeks {
			if _, valid := weekStart(name); valid && weeks[name] == nil {
				txn.Remove = append(txn.Remove, weekMetadata(name))
			}
		}
	}
	for name, data := range additional {
		txn.Writes[name] = data
	}
	encoded, err := json.Marshal(txn)
	if err != nil {
		return err
	}
	if err := atomicFile(root, journalName, encoded); err != nil {
		return err
	}
	return replay(root, txn)
}
