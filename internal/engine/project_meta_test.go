package engine

import "testing"

func TestProjectMetadataIsEditableWithoutRenamingTheFolder(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	a.touchProject(root)

	updated, err := a.UpdateProjectMeta(root, "หน้าร้านใหม่", "งานปรับ checkout และหน้าชำระเงิน")
	if err != nil {
		t.Fatalf("UpdateProjectMeta: %v", err)
	}
	if updated.Name != "หน้าร้านใหม่" || updated.Description != "งานปรับ checkout และหน้าชำระเงิน" {
		t.Fatalf("metadata was not saved: %#v", updated)
	}
	if updated.RootPath != root {
		t.Fatalf("editing metadata renamed the folder from %q to %q", root, updated.RootPath)
	}

	// Opening it again refreshes only recency/path; the user's words survive.
	a.touchProject(root)
	projects := a.RecentProjects()
	if len(projects) != 1 || projects[0].Name != "หน้าร้านใหม่" || projects[0].Description == "" {
		t.Fatalf("touchProject overwrote user metadata: %#v", projects)
	}
}

func TestProjectMetadataNeedsARememberedProject(t *testing.T) {
	a := newJobApp(t)
	if _, err := a.UpdateProjectMeta(t.TempDir(), "ghost", ""); err == nil {
		t.Fatal("updated a project that is not in the index")
	}
}
