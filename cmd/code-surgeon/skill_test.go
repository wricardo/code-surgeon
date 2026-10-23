package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallSkill_CreatesDirectoriesThenUpdatesInPlace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "skills", skillName)

	path, existed, err := installSkill(dir)
	require.NoError(t, err)
	require.False(t, existed)
	require.Equal(t, filepath.Join(dir, "SKILL.md"), path)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, skillMD, got)

	// a locally edited copy is replaced by the version embedded in the binary
	require.NoError(t, os.WriteFile(path, []byte("stale"), 0o644))
	_, existed, err = installSkill(dir)
	require.NoError(t, err)
	require.True(t, existed)
	got, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, skillMD, got)
}

func TestInstallSkill_ReportsUnwritableTarget(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	_, _, err := installSkill(filepath.Join(blocker, "skill")) // a file where a directory is needed
	require.Error(t, err)
}
