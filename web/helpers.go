package web

import (
	"errors"
	"log"

	"github.com/hfurubotten/autograder/entities"
	"github.com/hfurubotten/autograder/game/points"
	"github.com/hfurubotten/autograder/game/trophies"
)

//TODO move this stuff to game package or somewhere else, possibly entities; e.g. user_score.go?

// DistributeScores is a helper function which update the scores on repos and users.
// This function will up also check if the supplied objects also implements the saver
// interface, and if so lock the writing and save the object when done.
// locking is handled internally.
func DistributeScores(score int, user points.SingleScorer, org points.MultiScorer) (err error) {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	usaver, issaver := user.(entities.Saver)

	if issaver {
		usaver.Lock()
		defer func() {
			err = usaver.Save()
			if err != nil {
				usaver.Unlock()
				log.Println(err)
			}
		}()
	}

	user.IncScoreBy(score)

	if org != nil {
		osaver, issaver := org.(entities.Saver)

		if issaver {
			osaver.Lock()
			defer func() {
				err = osaver.Save()
				if err != nil {
					osaver.Unlock()
					log.Println(err)
				}
			}()
		}

		org.IncScoreBy(user.GetUsername(), score)
	}

	return
}

func RegisterAction(action int, user trophies.TrophyHunter) (err error) {
	usaver, issaver := user.(entities.Saver)

	if issaver {
		usaver.Lock()
		defer func() {
			err = usaver.Save()
			if err != nil {
				return
			}
		}()
	}
	chest := user.GetTrophyChest()

	trophy, ok := chest.Store[action]
	if !ok {
		trophy = trophies.StandardThrophyChest.Store[action]
		chest.Store[action] = trophy
	}

	trophy.Occurrences++
	trophy.BumpRank()

	return
}
