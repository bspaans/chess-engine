package chess_engine

import (
	"testing"
)

func Test_Eval_mate_white(t *testing.T) {

	cases := []string{
		"rn2k2r/1p3ppp/2p5/1p2p3/2P1n1bP/P5P1/4p2R/b1B1K1q1 w kq - 36 1",
		"1nb1k1nr/1p3ppp/2p5/3pp3/KpP1P1PP/q4P2/P1P5/5B1R w k - 36 1",
		"rn2k2r/1p3ppp/2p5/1p2p3/2P1n1bP/P5P1/4p2R/b1B1K1q1 w kq - 36 1",
		"r3kb1r/pp3ppp/2n2n2/3p4/Pq3pbP/1P2pK2/1BPPP1P1/RN1Q1B2 w kq - 22 12",
	}
	unit := Evaluators([]Evaluator{})
	for _, expected := range cases {
		position, err := ParseFEN(expected)
		if err != nil {
			t.Fatal(err)
		}
		if unit.Eval(position) != Mate {
			t.Errorf("Expecting mate")
		}
	}
}

func Test_Eval_mate_black(t *testing.T) {

	cases := []string{
		"r4b2/p3pB2/3N4/6Q1/6kp/P1N1B3/1PP2PPP/R3K2R b KQ - 45 1",
		"r4b2/p3pB2/3N4/6Q1/6kp/P1N1B3/1PP2PPP/R3K2R b KQ - 45 1",
	}
	unit := Evaluators([]Evaluator{})
	for _, expected := range cases {
		position, err := ParseFEN(expected)
		if err != nil {
			t.Fatal(err)
		}
		if unit.Eval(position) != Mate {
			t.Errorf("Expecting mate")
		}
	}
}

func Test_Eval_BestMove_white(t *testing.T) {

	cases := []string{
		"8/8/8/qn6/kn6/1n6/1KP5/8 w - - 0 0",
		"8/8/8/qn6/kn6/1n6/1KP5/1QQQQQQR w - - 0 0",
	}
	unit := Evaluators([]Evaluator{})
	for _, expected := range cases {
		position, err := ParseFEN(expected)
		if err != nil {
			t.Fatal(err)
		}
		nextGame, score := unit.BestMove(position)
		if !nextGame.IsMate() || score != Mate {
			t.Errorf("Expecting mate after best move")
		}
	}
}

func Test_Eval_BestMove_black(t *testing.T) {

	cases := []string{
		"8/1kp5/1N6/KN6/QN6/8/8/8 b - - 0 0",
	}
	unit := Evaluators([]Evaluator{})
	for _, expected := range cases {
		position, err := ParseFEN(expected)
		if err != nil {
			t.Fatal(err)
		}
		nextGame, score := unit.BestMove(position)
		if !nextGame.IsMate() && score != OpponentMate {
			t.Errorf("Expecting mate after best move")
		}
	}
}

func Test_Eval_BestMove_space_evaluator(t *testing.T) {
	unit := Evaluators([]Evaluator{SpaceEvaluator})
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	position, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}

	game, _ := unit.BestMove(position)
	if game.Line[0].String() != "e2e4" {
		t.Errorf("Expecting e2e4 as opening move for space evaluator, got %s", game.Line)
	}
}

func Test_Eval_BestLine_opening_space_evaluator(t *testing.T) {

	unit := Evaluators([]Evaluator{SpaceEvaluator})
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	position, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}

	line := unit.BestLine(position, 2)
	if len(line) != 3 {
		t.Fatalf("Expecting line of length 2+1, got %d", len(line))
	}
	if line[0] != position {
		t.Errorf("Expecting starting position as first element in the line")
	}
	if line[1].Line[0].String() != "e2e4" {
		t.Errorf("Expecting e2e4 as opening move for space evaluator, got %s", line[1].Line)
	}
	if line[2].Line[1].String() != "e7e5" {
		t.Errorf("Expecting e7e5 as opening reply move for space evaluator, got %s", line[2].Line)
	}
}
