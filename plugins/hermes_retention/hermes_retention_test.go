package main

import "testing"

func TestComputeCutoff(t *testing.T) {
	cases := []struct {
		name            string
		tipEpoch        uint64
		retentionEpochs uint64
		wantCutoff      uint64
		wantPurge       bool
	}{
		{
			name:            "tip below retention => no purge, no underflow",
			tipEpoch:        100,
			retentionEpochs: 2185,
			wantCutoff:      0,
			wantPurge:       false,
		},
		{
			name:            "tip equals retention => cutoff is 1 (keep [1..N])",
			tipEpoch:        2185,
			retentionEpochs: 2185,
			wantCutoff:      1,
			wantPurge:       true,
		},
		{
			name:            "keeps exactly retentionEpochs epochs (off-by-one regression)",
			tipEpoch:        100,
			retentionEpochs: 10,
			wantCutoff:      91, // DELETE WHERE epoch < 91 keeps 91..100 = 10 epochs
			wantPurge:       true,
		},
		{
			name:            "production-like values",
			tipEpoch:        61908,
			retentionEpochs: 2185,
			wantCutoff:      59724,
			wantPurge:       true,
		},
		{
			name:            "zero tip => no purge",
			tipEpoch:        0,
			retentionEpochs: 2185,
			wantCutoff:      0,
			wantPurge:       false,
		},
		{
			name:            "retention=1 => cutoff = tip (keep only tip)",
			tipEpoch:        100,
			retentionEpochs: 1,
			wantCutoff:      100,
			wantPurge:       true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCutoff, gotPurge := computeCutoff(tc.tipEpoch, tc.retentionEpochs)
			if gotCutoff != tc.wantCutoff || gotPurge != tc.wantPurge {
				t.Fatalf("computeCutoff(%d,%d) = (%d,%v); want (%d,%v)",
					tc.tipEpoch, tc.retentionEpochs, gotCutoff, gotPurge, tc.wantCutoff, tc.wantPurge)
			}
		})
	}
}
