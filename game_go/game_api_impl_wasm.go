//go:build js || wasip1

package main

var game *Game

//go:wasmexport game_init
func game_init() int32 {
	game = newGame()
	if game == nil {
		return -1
	}
	return 0
}

//go:wasmexport game_frame
func game_frame(g int32, dt float64) int32 {
	if g != 0 {
		return -1
	}
	return game.update(float32(dt))
}

//go:wasmexport game_event
func game_event(g int32,
	evType int32, keyOrBtn int32,
	isDown int32, isRepeat int32,
	mouseX float32, mouseY float32,
	scrollX float32, scrollY float32,
	winW int32, winH int32) int32 {
	if g != 0 {
		return -1
	}
	return game.handleEvent(
		evType, keyOrBtn,
		isDown, isRepeat,
		mouseX, mouseY,
		scrollX, scrollY,
		winW, winH)
}

//go:wasmexport game_get_score
func game_get_score(player int32) int32 {
	if game == nil || int(player) >= game.board.NumPlayers {
		return 0
	}
	return int32(game.board.Scores[player])
}

//go:wasmexport game_get_current_player
func game_get_current_player() int32 {
	if game == nil {
		return 0
	}
	return int32(game.board.CurrentPlayer)
}

//go:wasmexport game_get_num_players
func game_get_num_players() int32 {
	if game == nil {
		return 0
	}
	return int32(game.board.NumPlayers)
}

//go:wasmexport game_cleanup
func game_cleanup(g int32) int32 {
	if g != 0 {
		return -1
	}
	if game == nil {
		return 0
	}
	result := game.cleanup()
	game = nil
	return result
}
