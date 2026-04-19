// Offline replay test. Boots a game by feeding the reducer a sequence
// of realistic envelopes and asserts the derived state matches what a
// connected device would render. Run with:
//
//     node --test --experimental-strip-types src/store.test.ts
//
// Node 22's built-in test runner is enough — we don't need vitest for
// a single file.

import assert from "node:assert/strict";
import { describe, it } from "node:test";

import {
  currentPresident,
  initialState,
  reducer,
  seatedPlayers,
} from "./store.ts";
import type { Envelope, Game, Player } from "./types.ts";

function envelope<T extends Envelope>(env: T): { type: "envelope"; envelope: T } {
  return { type: "envelope", envelope: env };
}

function mkPlayer(partial: Partial<Player> & Pick<Player, "playerId" | "seat" | "displayName">): Player {
  return {
    gameId: "g1",
    userId: partial.displayName,
    isHost: false,
    isAlive: true,
    isConnected: true,
    createdAt: "",
    updatedAt: "",
    ...partial,
  } as Player;
}

function mkGame(): Game {
  return {
    gameId: "g1",
    joinCode: "ABCDE",
    hostUserId: "user-host",
    status: "lobby",
    phase: "lobby",
    round: 0,
    presidentSeat: 0,
    liberalPoliciesEnacted: 0,
    fascistPoliciesEnacted: 0,
    electionTracker: 0,
    vetoUnlocked: false,
    playerCount: 0,
    createdAt: "",
    updatedAt: "",
  };
}

describe("reducer", () => {
  it("applies a lobby snapshot", () => {
    const game = mkGame();
    const me = mkPlayer({ playerId: "p1", seat: 0, displayName: "Alice", isHost: true });
    const state = reducer(initialState, {
      type: "snapshot",
      game,
      players: [me],
      mePlayerId: "p1",
    });
    assert.equal(state.game?.gameId, "g1");
    assert.equal(state.me.playerId, "p1");
    assert.deepEqual(seatedPlayers(state).map((p) => p.playerId), ["p1"]);
  });

  it("ingests game_started and updates phase + seat", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: mkGame(),
      players: [mkPlayer({ playerId: "p1", seat: 0, displayName: "Alice" })],
      mePlayerId: "p1",
    });
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e", gameId: "g1", type: "game_started", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: { playerCount: 5, initialPresidentSeat: 2 },
    }));
    assert.equal(s.game?.phase, "nomination");
    assert.equal(s.game?.playerCount, 5);
    assert.equal(s.game?.presidentSeat, 2);
  });

  it("stores only the local player's whispered role", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: mkGame(),
      players: [mkPlayer({ playerId: "p1", seat: 0, displayName: "Alice" })],
      mePlayerId: "p1",
    });
    // Whisper addressed to p1 → populates me.
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e1", gameId: "g1", type: "roles_assigned", createdAt: "", updatedAt: "" },
      audience: { scope: "player", playerId: "p1" },
      payload: { role: "hitler", party: "fascist", teammates: [] },
    }));
    assert.equal(s.me.role, "hitler");
    // Whisper addressed to someone else → ignored.
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e2", gameId: "g1", type: "roles_assigned", createdAt: "", updatedAt: "" },
      audience: { scope: "player", playerId: "p2" },
      payload: { role: "liberal", party: "liberal" },
    }));
    assert.equal(s.me.role, "hitler");
  });

  it("tracks vote fingerprints without revealing choices until result lands", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: { ...mkGame(), phase: "election", currentGovernmentId: "gov1" },
      players: [
        mkPlayer({ playerId: "p1", seat: 0, displayName: "Alice" }),
        mkPlayer({ playerId: "p2", seat: 1, displayName: "Bob" }),
      ],
      mePlayerId: "p1",
    });
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e", gameId: "g1", type: "vote_cast", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: { governmentId: "gov1", playerId: "p2" },
    }));
    assert.equal(s.votesCast["p2"], true);
    assert.equal(Object.keys(s.votes).length, 0); // no choices yet

    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e2", gameId: "g1", type: "election_result", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: {
        governmentId: "gov1",
        passed: true,
        jaVotes: 2,
        neinVotes: 0,
        votes: { p1: "ja", p2: "ja" },
        reason: "all_voted",
      },
    }));
    assert.deepEqual(s.votes, { p1: "ja", p2: "ja" });
  });

  it("updates board counters on enactment and clears tracker", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: {
        ...mkGame(),
        electionTracker: 2,
        fascistPoliciesEnacted: 2,
      },
      players: [mkPlayer({ playerId: "p1", seat: 0, displayName: "A" })],
      mePlayerId: "p1",
    });
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e", gameId: "g1", type: "chancellor_enacted", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: {
        governmentId: "gov1",
        policy: "fascist",
        liberalPoliciesEnacted: 0,
        fascistPoliciesEnacted: 3,
      },
    }));
    assert.equal(s.fascist, 3);
    assert.equal(s.tracker, 0);
  });

  it("marks a player dead on execution", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: mkGame(),
      players: [
        mkPlayer({ playerId: "p1", seat: 0, displayName: "A" }),
        mkPlayer({ playerId: "p2", seat: 1, displayName: "B" }),
      ],
      mePlayerId: "p1",
    });
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e", gameId: "g1", type: "player_executed", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: { playerId: "p2", wasHitler: false, executedBySeat: 0 },
    }));
    assert.equal(s.players["p2"].isAlive, false);
    assert.equal(s.players["p1"].isAlive, true);
  });

  it("records the winner on game_ended", () => {
    let s = reducer(initialState, {
      type: "snapshot",
      game: mkGame(),
      players: [mkPlayer({ playerId: "p1", seat: 0, displayName: "A" })],
      mePlayerId: "p1",
    });
    s = reducer(s, envelope({
      gameId: "g1",
      event: { eventId: "e", gameId: "g1", type: "game_ended", createdAt: "", updatedAt: "" },
      audience: { scope: "broadcast" },
      payload: { winner: "liberal", winCondition: "hitler_executed" },
    }));
    assert.equal(s.winner, "liberal");
    assert.equal(s.winCondition, "hitler_executed");
    assert.equal(s.game?.status, "completed");
  });

  it("resolves the current president from seat rotation", () => {
    const s = reducer(initialState, {
      type: "snapshot",
      game: { ...mkGame(), presidentSeat: 2, phase: "nomination" },
      players: [
        mkPlayer({ playerId: "p1", seat: 0, displayName: "A" }),
        mkPlayer({ playerId: "p2", seat: 1, displayName: "B" }),
        mkPlayer({ playerId: "p3", seat: 2, displayName: "C" }),
      ],
      mePlayerId: "p1",
    });
    assert.equal(currentPresident(s)?.playerId, "p3");
  });
});
