package entities

import (
	"encoding/gob"
	"sync"
	"time"

	"github.com/hfurubotten/autograder/game/levels"
	"github.com/hfurubotten/autograder/game/trophies"
)

func init() {
	gob.Register(UserScore{})
}

// UserScore keep track of the scores for a user.
type UserScore struct {
	lock *sync.RWMutex

	TotalScore   int64
	WeeklyScore  map[int]int64
	MonthlyScore map[time.Month]int64
	Level        int
	Trophies     *trophies.TrophyChest
}

// NewUserScore returns a new user score object.
func NewUserScore() (uc *UserScore) {
	uc = &UserScore{
		lock:         &sync.RWMutex{},
		WeeklyScore:  make(map[int]int64),
		MonthlyScore: make(map[time.Month]int64),
	}

	// if uc.WeeklyScore == nil {
	// 	uc.WeeklyScore = make(map[int]int64)
	// }
	// if uc.MonthlyScore == nil {
	// 	uc.MonthlyScore = make(map[time.Month]int64)
	// }
	return
}

// IncScoreBy increases the total score with given amount.
func (u *UserScore) IncScoreBy(score int) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.TotalScore += int64(score)
	u.Level = levels.FindLevel(u.TotalScore) // How to tackle level up notification?

	_, week := time.Now().ISOWeek()
	month := time.Now().Month()

	// updates weekly
	u.WeeklyScore[week] += int64(score)
	// updated monthly
	u.MonthlyScore[month] += int64(score)
}

// DecScoreBy descreases the total score with given amount.
func (u *UserScore) DecScoreBy(score int) {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.TotalScore-int64(score) > 0 {
		u.TotalScore -= int64(score)
	} else {
		u.TotalScore = 0
	}

	u.Level = levels.FindLevel(u.TotalScore)

	_, week := time.Now().ISOWeek()
	month := time.Now().Month()

	// updates weekly
	u.WeeklyScore[week] -= int64(score)
	// updated monthly
	u.MonthlyScore[month] -= int64(score)
}

// IncLevel increases the level with one.
func (u *UserScore) IncLevel() {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.Level++
}

// DecLevel decreases the level with one until it equals zero.
func (u *UserScore) DecLevel() {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.Level > 0 {
		u.Level--
	}
}

// GetTrophyChest return the users ThropyChest.
func (u *UserScore) GetTrophyChest() *trophies.TrophyChest {
	if u.Trophies == nil {
		u.Trophies = trophies.NewTrophyChest()
	}

	return u.Trophies
}
