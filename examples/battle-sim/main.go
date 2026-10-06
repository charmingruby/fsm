package main

import (
	"context"
	"fmt"
	"log"

	"github.com/charmingruby/fsm/fsm"
)

const (
	joinBattleState   fsm.State = "joinBattle"
	chooseActionState fsm.State = "chooseAction"
	fightState        fsm.State = "fight"
	retreatState      fsm.State = "retreat"
	victoryState      fsm.State = "victory"
	defeatedState     fsm.State = "defeated"
)

type player struct {
	race   string
	hp     int
	attack int
	speed  int
}

type gameData struct {
	you     player
	enemy   player
	actions []string
	turn    int
}

func main() {
	m := fsm.New[gameData](joinBattleState, fsm.WithLogger[gameData](fsm.NewStdLogger()))

	m.
		On(joinBattleState, func(_ context.Context, data *gameData) (fsm.State, error) {
			return chooseActionState, nil
		}, fsm.StateHooks[gameData]{
			OnEnter: func(ctx context.Context, data *gameData, state fsm.State) {
				fmt.Printf("%s(you) vs %s", data.you.race, data.enemy.race)
			},
		}).
		On(chooseActionState, func(_ context.Context, data *gameData) (fsm.State, error) {
			action := "fight"
			if data.turn < len(data.actions) {
				action = data.actions[data.turn]
			}

			data.turn++

			fmt.Printf("[turn %d] action: %s (HP %s=%d %s=%d)\n",
				data.turn, action, data.you.race, data.you.hp, data.enemy.race, data.enemy.hp)

			if action == "retreat" {
				return retreatState, nil
			}

			return fightState, nil
		}).
		On(fightState, func(_ context.Context, data *gameData) (fsm.State, error) {
			first, second := &data.you, &data.enemy
			firstIsPlayer := data.you.speed >= data.enemy.speed

			if !firstIsPlayer {
				first, second = second, first
			}

			strike(first, second)
			if second.hp <= 0 {
				return resultState(firstIsPlayer), nil
			}

			strike(second, first)
			if first.hp <= 0 {
				return resultState(!firstIsPlayer), nil
			}

			return chooseActionState, nil
		}).
		Terminal(
			retreatState,
			victoryState,
			defeatedState,
		)

	if err := m.Validate(); err != nil {
		log.Fatal(err)
	}

	data := &gameData{
		you:     player{race: "Elf", hp: 30, attack: 7, speed: 10},
		enemy:   player{race: "Human", hp: 24, attack: 6, speed: 8},
		actions: []string{"fight", "fight", "fight", "fight"},
	}

	trace, err := m.Run(context.Background(), data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("battle over after %d turns: %s=%d %s=%d trace=%v\n",
		data.turn, data.you.race, data.you.hp, data.enemy.race, data.enemy.hp, trace)
}

func strike(attacker, defender *player) {
	defender.hp -= attacker.attack
	fmt.Printf("%s (spd %d) hits %s for %d damage (%s HP=%d)\n",
		attacker.race, attacker.speed, defender.race, attacker.attack, defender.race, defender.hp)
}

func resultState(playerWon bool) fsm.State {
	if playerWon {
		return victoryState
	}

	return defeatedState
}
