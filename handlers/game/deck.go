package game

import (
	"context"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// initDeck builds a fresh deck per the game's RulesConfig (6 human / 11
// AI in the vanilla ruleset) and shuffles it in place with the engine's
// RNG.
func (h *GameHandler) initDeck(g *models.Game) {
	total := g.Rules.HumanProtocolsInDeck + g.Rules.AIProtocolsInDeck
	deck := make([]secrethitler.PolicyType, 0, total)
	for i := 0; i < g.Rules.HumanProtocolsInDeck; i++ {
		deck = append(deck, secrethitler.PolicyHuman)
	}
	for i := 0; i < g.Rules.AIProtocolsInDeck; i++ {
		deck = append(deck, secrethitler.PolicyAI)
	}
	h.rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	g.DrawPile = deck
	g.DiscardPile = nil
}

// drawThree takes the top three cards off the draw pile, reshuffling
// from the discard pile if needed. A DeckReshuffled event is emitted
// whenever a reshuffle occurs. Panics if the total deck has fewer than
// three remaining policies (the deck is replenished with enacted
// policies subtracted, which is always >= 3 in a legal game).
func (h *GameHandler) drawThree(ctx context.Context, g *models.Game) []secrethitler.PolicyType {
	if len(g.DrawPile) < 3 {
		h.reshuffle(ctx, g)
	}
	drawn := append([]secrethitler.PolicyType(nil), g.DrawPile[:3]...)
	g.DrawPile = g.DrawPile[3:]
	return drawn
}

// reshuffle merges draw + discard, shuffles, and emits an event.
func (h *GameHandler) reshuffle(ctx context.Context, g *models.Game) {
	combined := append([]secrethitler.PolicyType{}, g.DrawPile...)
	combined = append(combined, g.DiscardPile...)
	h.rng.Shuffle(len(combined), func(i, j int) { combined[i], combined[j] = combined[j], combined[i] })
	g.DrawPile = combined
	g.DiscardPile = nil

	ev := models.NewGameEvent(g.GameID, secrethitler.EventDeckReshuffled, "")
	h.broadcast(ctx, ev, DeckReshuffledPayload{RemainingInDraw: len(g.DrawPile)})
}

// discardPolicy places a policy onto the discard pile.
func discardPolicy(g *models.Game, p secrethitler.PolicyType) {
	g.DiscardPile = append(g.DiscardPile, p)
}
