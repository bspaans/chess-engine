package chess_engine

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"time"
)

// Uses depth first search

type DFSEngine struct {
	StartingPosition *FEN
	Cancel           context.CancelFunc
	Evaluators       []Evaluator
	EvalTree         *EvalTree
	SelDepth         int
}

func NewDFSEngine(depth int) *DFSEngine {
	return &DFSEngine{
		SelDepth: depth,
	}
}

func (b *DFSEngine) SetPosition(fen *FEN) {
	b.StartingPosition = fen
}

func (b *DFSEngine) SetOption(opt EngineOption, val int) {
	if opt == SELDEPTH {
		b.SelDepth = val
	}
}

func (b *DFSEngine) Start(output chan string, maxNodes, maxDepth int) {
	ctx, cancel := context.WithCancel(context.Background())
	b.Cancel = cancel
	go b.start(ctx, output, maxNodes, maxDepth)
}

func (b *DFSEngine) start(ctx context.Context, output chan string, maxNodes, maxDepth int) {
	seen := map[string]bool{}
	b.EvalTree = NewEvalTree(nil, math.Inf(-1))
	timer := time.NewTimer(time.Second)
	depth := b.SelDepth + 1
	nodes := 0
	totalNodes := 0
	var bestLine *EvalTree

	firstLine := b.InitialBestLine(b.SelDepth)
	queue := list.New()
	lineLength := 0
	for _, m := range firstLine {
		if m != nil {
			fenStr := m.FENString()
			seen[fenStr] = true
		}
	}
	for d := 0; d < b.SelDepth; d++ {
		if firstLine[d] != nil {
			lineLength++
			queue.PushFront(firstLine[d])
		}
	}
	// Queue all the other positions from the starting position
	nextFENs := b.StartingPosition.NextFENs()
	for _, f := range nextFENs {
		if f.Line[0].String() != firstLine[0].Line[0].String() {
			// Skip uninteresting moves
			if !b.ShouldCheckPosition(f) {
				continue
			}
			queue.PushBack(f)
		}
	}

	for {
		select {
		case <-ctx.Done():
			output <- fmt.Sprintf("bestmove %s", b.EvalTree.BestLine.Move.String())
			goto end
		case <-timer.C:
			totalNodes += nodes
			output <- fmt.Sprintf("info ns %d nodes %d depth %d queue %d", nodes, totalNodes, depth, queue.Len())
			nodes = 0
			timer = time.NewTimer(time.Second)
			bestLine = b.EvalTree.BestLine
			bestResult := bestLine.GetBestLine()
			line := Line(bestResult.Line).String()
			output <- fmt.Sprintf("info depth %d score cp %d pv %s", len(bestResult.Line), int(math.Round(bestResult.Score*100)), line)
		default:
			if queue.Len() > 0 {
				nodes++
				game := queue.Remove(queue.Front()).(*FEN)

				if len(game.Line) < depth {
					b.EvalTree.UpdateBestLine()
					//b.EvalTree.Prune()
				}
				depth = len(game.Line)
				fenStr := game.FENString()
				seen[fenStr] = true

				score := 0.0
				if game.IsDraw() {
					score = 0.0
				} else if game.IsMate() {
					score = 58008
				} else {
					score = b.heuristicScorePosition(game)
				}

				b.EvalTree.Insert(game.Line, score)

				if len(game.Line) < b.SelDepth {
					nextFENs := game.NextFENs()
					wasForced := len(nextFENs) == 1
					for _, f := range nextFENs {
						// Skip "uninteresting" moves
						if !wasForced && !b.ShouldCheckPosition(f) {
							continue
						}
						if !seen[f.FENString()] {
							queue.PushFront(f)
						}
					}
				}
				if maxNodes > 0 && totalNodes+nodes >= maxNodes {
					output <- fmt.Sprintf("info ns %d nodes %d depth %d", nodes, totalNodes, depth)
					output <- fmt.Sprintf("bestmove %s", b.EvalTree.BestLine.Move.String())
					return
				}
			} else {
				bestLine = b.EvalTree.BestLine
				bestResult := bestLine.GetBestLine()
				line := Line(bestResult.Line).String()
				output <- fmt.Sprintf("info depth %d score cp %d pv %s", len(bestResult.Line), int(math.Round(bestResult.Score*100)), line)
				output <- fmt.Sprintf("info ns %d nodes %d depth %d", nodes, totalNodes, depth)
				output <- fmt.Sprintf("bestmove %s", b.EvalTree.BestLine.Move.String())
				goto end
			}
		}
	}
end:
}

func (b *DFSEngine) ShouldCheckPosition(position *FEN) bool {
	valid := position.ValidMoves()

	/*
			TODO: enable this when we can shortcut the searchtree for Mate in Ns; otherwise this makes the tests blow up
		attacks := position.Attacks.GetAttacks(position.ToMove, position.Pieces)
		validAttacks := position.FilterPinnedPieces(attacks)
				// Look at all the moves leading to checks
				for _, m := range valid {
					if position.ApplyMove(m).InCheck() {
						return true
					}
				}
	*/
	return position.InCheck() || len(valid) <= 1 //|| len(validAttacks) > 0
}

func (b *DFSEngine) InitialBestLine(depth int) []*FEN {
	line := make([]*FEN, depth)
	game := b.StartingPosition
	for d := 0; d < depth; d++ {
		move, gameFinished := b.BestMove(game)
		if move != nil {
			game = game.ApplyMove(move)
			line[d] = game
			if gameFinished {
				break
			}
		} else {
			break
		}
	}
	return line
}

func (b *DFSEngine) BestMove(game *FEN) (*Move, bool) {
	nextFENs := game.NextFENs()
	bestScore := math.Inf(-1)
	var bestGame *FEN
	var bestMove *Move

	for _, f := range nextFENs {
		score := math.Inf(-1)
		if f.IsDraw() {
			score = 0.0
		} else if f.IsMate() {
			score = math.Inf(1)
		} else {
			score = b.heuristicScorePosition(f) * -1
		}
		if score > bestScore {
			bestScore = score
			bestGame = f
			bestMove = f.Line[len(f.Line)-1]
		}
	}
	b.EvalTree.Insert(append(game.Line, bestMove), bestScore)
	return bestMove, bestGame.IsDraw() || bestGame.IsMate()
}

func (b *DFSEngine) AddEvaluator(e Evaluator) {
	b.Evaluators = append(b.Evaluators, e)
}

func (b *DFSEngine) heuristicScorePosition(f *FEN) float64 {
	score := 0.0
	for _, eval := range b.Evaluators {
		score += eval(f)
	}
	if f.ToMove == Black {
		return score * -1
	}
	return score
}

func (b *DFSEngine) Stop() {
	b.Cancel()
}
