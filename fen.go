package chess_engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Color int8

const (
	NoColor Color = iota
	Black
	White
)

func (c Color) String() string {
	if c == White {
		return "w"
	} else if c == Black {

		return "b"
	}
	return " "
}

func (c Color) Opposite() Color {
	if c == Black {
		return White
	} else {
		return Black
	}
}

type CastleStatus int8

const (
	Both CastleStatus = iota
	None
	Kingside
	Queenside
)

func (cs CastleStatus) String(c Color) string {
	type p struct {
		CastleStatus
		Color
	}
	switch (p{cs, c}) {
	case p{Both, Black}:
		return "kq"
	case p{Both, White}:
		return "KQ"
	case p{Kingside, Black}:
		return "k"
	case p{Kingside, White}:
		return "K"
	case p{Queenside, Black}:
		return "q"
	case p{Queenside, White}:
		return "Q"
	}
	if cs == None {
		return "-"
	}
	return ""
}

type FEN struct {
	// An array of size 64 denoting the board.
	// 0 index = a1
	Board []Piece
	// The location of every piece on the board.
	// The Pieces are normalized, because the color
	// is already part of the map.
	Pieces PiecePositions

	ToMove              Color
	WhiteCastleStatus   CastleStatus
	BlackCastleStatus   CastleStatus
	EnPassantVulnerable Position
	HalfmoveClock       int
	Fullmove            int

	// The line we're currently pondering on
	Line []*Move
}

func ParseFEN(fenstr string) (*FEN, error) {
	fen := FEN{}
	fen.BlackCastleStatus = None
	fen.WhiteCastleStatus = None
	forStr := ""
	colorStr := ""
	castleStr := ""
	enPassant := ""
	_, err := fmt.Sscanf(fenstr, "%s %s %s %s %d %d",
		&forStr,
		&colorStr,
		&castleStr,
		&enPassant,
		&fen.HalfmoveClock,
		&fen.Fullmove,
	)
	if err != nil {
		return nil, err
	}
	switch colorStr {
	case "w":
		fen.ToMove = White
	case "b":
		fen.ToMove = Black
	default:
		return nil, errors.New("pgn: invalid color")
	}

	if strings.Contains(castleStr, "k") {
		fen.BlackCastleStatus = Kingside
	}
	if strings.Contains(castleStr, "q") {
		if fen.BlackCastleStatus == Kingside {
			fen.BlackCastleStatus = Both
		} else {
			fen.BlackCastleStatus = Queenside
		}
	}

	if strings.Contains(castleStr, "K") {
		fen.WhiteCastleStatus = Kingside
	}
	if strings.Contains(castleStr, "Q") {
		if fen.WhiteCastleStatus == Kingside {
			fen.WhiteCastleStatus = Both
		} else {
			fen.WhiteCastleStatus = Queenside
		}
	}

	if enPassant == "-" {
		fen.EnPassantVulnerable = NoPosition
	} else {
		fen.EnPassantVulnerable, err = ParsePosition(enPassant)
		if err != nil {
			return nil, err
		}
	}
	fen.Board = make([]Piece, 64)
	for i := 0; i < 64; i++ {
		fen.Board[i] = NoPiece
	}
	fen.Pieces = NewPiecePositions()
	x := 0
	y := 7
	for i := 0; i < len(forStr); i++ {
		// if we're at the end of the row
		if forStr[i] == '/' {
			x = 0
			y--
		} else if forStr[i] >= '1' && forStr[i] <= '8' {
			// if we have blank squares
			j, err := strconv.Atoi(string(forStr[i]))
			if err != nil {
				return nil, err
			}
			x += j
		} else {
			// if we have a piece
			pos := y*8 + x
			piece := Piece(forStr[i])
			fen.Board[pos] = piece
			fen.Pieces.AddPosition(piece, Position(pos))
			x++
		}
	}
	return &fen, nil
}

// Returns new FENs for every valid move from the current FEN
func (f *FEN) NextFENs() []*FEN {
	moves := f.ValidMoves()
	result := []*FEN{}
	for _, m := range moves {
		result = append(result, f.ApplyMove(m))
	}
	return result
}

func (f *FEN) GetAttacksOnCondition(cond func(p Position) bool, color Color) []*Move {

	result := []*Move{}

	for _, pawnPos := range f.Pieces.Positions(color, Pawn) {
		positions := PawnAttacks[color][pawnPos]
		for _, p := range positions {
			if cond(p) {
				move := NewMove(pawnPos, p)
				// Handle promotions
				promotions := move.ToPromotions()
				if promotions == nil {
					result = append(result, move)
				} else {
					for _, m := range promotions {
						result = append(result, m)
					}
				}
			}
		}
		// TODO en passant
	}
	for _, piece := range []NormalizedPiece{Knight} {
		for _, fromPos := range f.Pieces.Positions(color, piece) {
			for _, toPos := range PieceMoves[Piece(piece)][fromPos] {
				if cond(toPos) {
					result = append(result, NewMove(fromPos, toPos))
				}
			}

		}
	}
	for _, piece := range []NormalizedPiece{Bishop, Rook, Queen} {
		for _, fromPos := range f.Pieces.Positions(color, piece) {
			for _, line := range MoveVectors[Piece(piece)][fromPos] {
				for _, toPos := range line {
					if cond(toPos) {
						result = append(result, NewMove(fromPos, toPos))
					} else if f.Board[toPos] == NoPiece {
						continue
					}
					break
				}
			}

		}
	}
	// TODO king attacks only if piece is undefended
	return result
}

func (f *FEN) GetAttacks(color Color) []*Move {
	cond := func(p Position) bool {
		return f.Board[p] != NoPiece && f.Board[p].Color() == color.Opposite()
	}
	return f.GetAttacksOnCondition(cond, color)
}
func (f *FEN) AttacksSquare(color Color, square Position) bool {
	cond := func(p Position) bool {
		return p == square
	}
	attacks := f.GetAttacksOnCondition(cond, color)
	/*
		fmt.Println(square, attacks)
		fmt.Println(f.Board[H2] == WhitePawn, f.Board[G2] == WhitePawn, f.Board[F2] == WhitePawn)
		fmt.Println(f.Pieces[White][Pawn])
		fmt.Println(f.Pieces)
	*/
	return len(attacks) > 0
}

func (f *FEN) GetIncomingAttacks() []*Move {
	return f.GetAttacks(f.ToMove.Opposite())
}

func (f *FEN) validMovesInCheck(checks []*Move) []*Move {
	result := []*Move{}
	// 1. move the king
	for _, kingPos := range f.Pieces.Positions(f.ToMove, King) {
		for _, p := range kingPos.GetKingMoves() {
			if (f.Board[p] == NoPiece || f.Board[p].Color() == f.ToMove.Opposite()) && !f.AttacksSquare(f.ToMove.Opposite(), p) {
				result = append(result, NewMove(kingPos, p))
			}
		}
	}
	// 2. block the attack
	// 3. remove the attacking piece
	if len(checks) == 1 {
		for _, check := range checks {
			// if the piece is a knight the check cannot be blocked
			attackingPiece := f.Board[check.From]
			if NormalizedPiece(attackingPiece.Normalize()) == Knight {
				break
			}
			diffFile := int(check.From.GetFile()) - int(check.To.GetFile())
			diffRank := int(check.From.GetRank()) - int(check.To.GetRank())
			maxDiff := diffFile
			if maxDiff < 0 {
				maxDiff = maxDiff * -1
			}
			if diffRank > maxDiff {
				maxDiff = diffRank
			} else if (diffRank * -1) > maxDiff {
				maxDiff = diffRank * -1
			}
			normDiffFile, normDiffRank := diffFile/maxDiff, diffRank/maxDiff
			blocks := map[Position]bool{}
			pos := check.To
			i := 0
			for pos != check.From {
				pos = Position(int(pos) + normDiffFile + (normDiffRank * 8))
				blocks[pos] = true
				i++
				if i > 7 {
					fmt.Println(checks)
					fmt.Println(string([]byte{byte(f.Board[pos])}))
					panic("wtf")
				}
			}
			cond := func(p Position) bool {
				return blocks[p]
			}
			for _, m := range f.GetAttacksOnCondition(cond, f.ToMove) {
				result = append(result, m)
			}
		}
	}
	return result
}

func (f *FEN) FENString() string {
	forStr := ""
	for y := 7; y >= 0; y-- {
		empty := 0
		for x := 0; x < 8; x++ {
			pos := y*8 + x
			if f.Board[pos] != NoPiece {
				if empty != 0 {
					forStr += strconv.Itoa(empty)
				}
				forStr += string([]byte{byte(f.Board[pos])})
				empty = 0
			} else {
				empty += 1
			}
		}
		if empty != 0 {
			forStr += strconv.Itoa(empty)
		}
		if y != 0 {
			forStr += "/"
		}
	}
	castleStatus := f.WhiteCastleStatus.String(White) + f.BlackCastleStatus.String(Black)
	if castleStatus == "--" {
		castleStatus = "-"
	}
	if castleStatus != "-" && strings.Contains(castleStatus, "-") {
		castleStatus = strings.Trim(castleStatus, "-")
	}
	enPassant := "-"
	if f.EnPassantVulnerable != 0 {
		enPassant = f.EnPassantVulnerable.String()
	}
	return fmt.Sprintf("%s %s %s %s %d %d", forStr, f.ToMove.String(), castleStatus, enPassant, f.HalfmoveClock, f.Fullmove)
}

func (f *FEN) IsMate() bool {
	incoming := f.GetIncomingAttacks()
	fmt.Println(incoming)
	checks := []*Move{}
	for _, attack := range incoming {
		if attack.To == f.Pieces.GetKingPos(f.ToMove) {
			checks = append(checks, attack)
		}
	}
	if len(checks) > 0 {
		moves := f.validMovesInCheck(checks)
		return len(moves) == 0
	} else {
		return false
	}
}

func (f *FEN) ValidMoves() []*Move {
	result := []*Move{}

	incoming := f.GetIncomingAttacks()
	fmt.Println(incoming)
	checks := []*Move{}
	for _, attack := range incoming {
		if attack.To == f.Pieces.GetKingPos(f.ToMove) {
			checks = append(checks, attack)
		}
	}
	if len(checks) > 0 {
		return f.validMovesInCheck(checks)
	}
	// TODO: make sure pieces aren't pinned

	for _, attack := range f.GetAttacks(f.ToMove) {
		result = append(result, attack)
	}

	for _, pawnPos := range f.Pieces.Positions(f.ToMove, Pawn) {
		for _, targetPos := range PieceMoves[f.Board[pawnPos]][pawnPos] {
			if f.Board[targetPos] == NoPiece {
				move := NewMove(pawnPos, targetPos)
				promotions := move.ToPromotions()
				if promotions == nil {
					result = append(result, move)
				} else {
					for _, m := range promotions {
						result = append(result, m)
					}
				}
			}
		}
	}
	for _, piece := range []NormalizedPiece{Knight} {
		for _, fromPos := range f.Pieces.Positions(f.ToMove, piece) {
			for _, toPos := range PieceMoves[Piece(piece)][fromPos] {
				if f.Board[toPos] == NoPiece {
					result = append(result, NewMove(fromPos, toPos))
				}
			}

		}
	}
	for _, piece := range []NormalizedPiece{Bishop, Rook, Queen} {
		for _, fromPos := range f.Pieces.Positions(f.ToMove, piece) {
			for _, line := range MoveVectors[Piece(piece)][fromPos] {
				for _, toPos := range line {
					if f.Board[toPos] == NoPiece {
						result = append(result, NewMove(fromPos, toPos))
					} else {
						break
					}
				}
			}

		}
	}
	kingPos := f.Pieces.GetKingPos(f.ToMove)
	for _, p := range kingPos.GetKingMoves() {
		// TODO only if p is not under attack
		if f.Board[p] == NoPiece {
			result = append(result, NewMove(kingPos, p))
		}
	}
	// TODO castling
	return result
}

func (f *FEN) ApplyMove(move *Move) *FEN {
	result := &FEN{}
	line := make([]*Move, len(f.Line)+1)
	for i, m := range f.Line {
		line[i] = m
	}
	line[len(f.Line)] = move

	board := make([]Piece, 64)
	for i := 0; i < 64; i++ {
		board[i] = f.Board[i]
	}
	movingPiece := board[move.From]
	board[move.From] = NoPiece
	capturedPiece := board[move.To].ToNormalizedPiece()
	board[move.To] = movingPiece
	normalizedMovingPiece := movingPiece.ToNormalizedPiece()

	if move.Promote != NoPiece {
		board[move.To] = move.Promote
	}

	wCastle := f.WhiteCastleStatus
	bCastle := f.BlackCastleStatus
	switch movingPiece {
	case BlackRook:
		switch move.From {
		case A8:
			switch bCastle {
			case Both:
				bCastle = Kingside
			case Queenside:
				bCastle = None
			}
		case H8:
			switch bCastle {
			case Both:
				bCastle = Queenside
			case Kingside:
				bCastle = None
			}
		}
	case BlackKing:
		// handle castles
		if move.From == E8 && move.To == G8 {
			if bCastle != Kingside && bCastle != Both {
				panic("Invalid castle")
			}
			// TODO: implement castle
		} else if move.From == E8 && move.To == C8 {
			if bCastle != Queenside && bCastle != Both {
				panic("Invalid castle")
			}
			// TODO: implement castle
		}
		bCastle = None
	case WhiteRook:
		switch move.From {
		case A1:
			switch wCastle {
			case Both:
				wCastle = Kingside
			case Queenside:
				wCastle = None
			}
		case H1:
			switch wCastle {
			case Both:
				wCastle = Queenside
			case Kingside:
				wCastle = None
			}
		}
	case WhiteKing:
		// handle castles
		if move.From == E1 && move.To == G1 {
			if wCastle != Kingside && wCastle != Both {
				panic("invalid castle")
			}
			// TODO handle castle
		} else if move.From == E1 && move.To == C1 {
			if wCastle != Queenside && wCastle != Both {
				panic("invalid castle")
			}
			// TODO handle castle
		}
		wCastle = None
	}
	pieces := f.Pieces.ApplyMove(f.ToMove, move, normalizedMovingPiece, capturedPiece)

	result.Board = board
	result.Pieces = pieces

	fullMove := f.Fullmove
	if f.ToMove == Black {
		fullMove += 1
	}

	result.ToMove = f.ToMove.Opposite()
	result.WhiteCastleStatus = wCastle
	result.BlackCastleStatus = bCastle
	result.EnPassantVulnerable = NoPosition // TODO
	result.HalfmoveClock = f.HalfmoveClock + 1
	result.Fullmove = fullMove
	result.Line = line
	return result
}
