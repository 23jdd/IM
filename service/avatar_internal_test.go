package service

import (
	"context"
	"errors"
	"testing"
)

func TestUploadAvatarStoresAndUpdates(t *testing.T) {
	origSave := saveImage
	origUpdate := updateUserAvatar
	defer func() { saveImage = origSave; updateUserAvatar = origUpdate }()

	var savedData []byte
	var savedCT string
	saveImage = func(ctx context.Context, data []byte, ct string) (string, error) {
		savedData = data
		savedCT = ct
		return "objid123", nil
	}
	var gotUid, gotAvatar string
	updateUserAvatar = func(ctx context.Context, uid, avatarId string) error {
		gotUid = uid
		gotAvatar = avatarId
		return nil
	}

	id, err := UploadAvatar(context.Background(), "u1", []byte("pngbytes"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if id != "objid123" {
		t.Errorf("id = %s, want objid123", id)
	}
	if string(savedData) != "pngbytes" || savedCT != "image/png" {
		t.Errorf("saveImage got data=%q ct=%s", savedData, savedCT)
	}
	if gotUid != "u1" || gotAvatar != "objid123" {
		t.Errorf("updateUserAvatar got uid=%s avatar=%s, want u1/objid123", gotUid, gotAvatar)
	}
}

func TestUploadAvatarSaveErrorSkipsUpdate(t *testing.T) {
	origSave := saveImage
	origUpdate := updateUserAvatar
	defer func() { saveImage = origSave; updateUserAvatar = origUpdate }()

	saveImage = func(ctx context.Context, data []byte, ct string) (string, error) {
		return "", errors.New("mongo down")
	}
	updateCalled := false
	updateUserAvatar = func(ctx context.Context, uid, avatarId string) error {
		updateCalled = true
		return nil
	}

	if _, err := UploadAvatar(context.Background(), "u1", []byte("x"), "image/png"); err == nil {
		t.Fatal("expected error when image save fails")
	}
	if updateCalled {
		t.Error("avatar update must not run when image save fails")
	}
}

func TestUploadAvatarUpdateErrorReturnsError(t *testing.T) {
	origSave := saveImage
	origUpdate := updateUserAvatar
	defer func() { saveImage = origSave; updateUserAvatar = origUpdate }()

	saveImage = func(ctx context.Context, data []byte, ct string) (string, error) { return "id", nil }
	updateUserAvatar = func(ctx context.Context, uid, avatarId string) error {
		return errors.New("db down")
	}

	if _, err := UploadAvatar(context.Background(), "u1", []byte("x"), "image/png"); err == nil {
		t.Fatal("expected error when avatar update fails")
	}
}

func TestGetAvatarByUidReturnsImage(t *testing.T) {
	origFind := findUserAvatar
	origGet := getImage
	defer func() { findUserAvatar = origFind; getImage = origGet }()

	findUserAvatar = func(ctx context.Context, uid string) (string, error) { return "id1", nil }
	getImage = func(ctx context.Context, id string) ([]byte, string, error) {
		if id != "id1" {
			t.Errorf("getImage id = %s, want id1", id)
		}
		return []byte("imgdata"), "image/jpeg", nil
	}

	data, ct, err := GetAvatarByUid(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "imgdata" || ct != "image/jpeg" {
		t.Errorf("got data=%q ct=%s", data, ct)
	}
}

func TestGetAvatarByUidNoAvatarSkipsImage(t *testing.T) {
	origFind := findUserAvatar
	origGet := getImage
	defer func() { findUserAvatar = origFind; getImage = origGet }()

	findUserAvatar = func(ctx context.Context, uid string) (string, error) { return "", nil }
	imageCalled := false
	getImage = func(ctx context.Context, id string) ([]byte, string, error) {
		imageCalled = true
		return nil, "", nil
	}

	data, _, err := GetAvatarByUid(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Errorf("expected nil data for no avatar, got %q", data)
	}
	if imageCalled {
		t.Error("getImage must not be called when user has no avatar")
	}
}

func TestGetAvatarByUidLookupErrorIsGraceful(t *testing.T) {
	origFind := findUserAvatar
	defer func() { findUserAvatar = origFind }()

	findUserAvatar = func(ctx context.Context, uid string) (string, error) {
		return "", errors.New("not found")
	}

	data, _, err := GetAvatarByUid(context.Background(), "ghost")
	if err != nil {
		t.Errorf("lookup error should be graceful (no error), got %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %q", data)
	}
}
