package main

// The deck renderer as the engine sees it (§248 B1): engine.Screen.RenderDeck.
// A deck is a file the engine names; what it looks like — printed, or slide
// by slide — is a question only a browser can answer, and the unseen WebView2
// tab that answers it is this window's (deck_render.go).

import (
	"context"
	"fmt"

	"github.com/Mikedev115/Aetox/internal/engine"
)

func (s appScreen) RenderDeck(ctx context.Context, fileURL string, req engine.DeckRender) (engine.DeckRendered, error) {
	switch req.Kind {
	case "pdf":
		pdf, err := s.app.exportDeckPDF(ctx, fileURL)
		return engine.DeckRendered{PDF: pdf}, err
	case "images":
		shots, err := s.app.exportDeckImages(ctx, fileURL, req.Format)
		return engine.DeckRendered{Images: shots}, err
	case "slide":
		shot, err := s.app.captureDeckSlide(ctx, fileURL, req.Slide)
		if err != nil {
			return engine.DeckRendered{}, err
		}
		return engine.DeckRendered{Images: [][]byte{shot}}, nil
	default:
		return engine.DeckRendered{}, fmt.Errorf("ไม่รู้จักการเรนเดอร์ %q", req.Kind)
	}
}

// captureDeckSlide is exportDeckImages for exactly one slide.
//
// The two passes are not optional and are not this file's to invent: revealing
// so a slide has its content at all, flattening so it has a place of its own to
// be clipped to. deck_image.go carries the reasoning for both.
func (a *App) captureDeckSlide(ctx context.Context, fileURL string, slide int) ([]byte, error) {
	spec, ok := deckImageFormats["png"]
	if !ok {
		return nil, fmt.Errorf("ไม่มีตัวเขียนภาพ png")
	}
	var shot []byte
	err := a.withExportTab(ctx, fileURL, func(call engineCaller) error {
		if _, err := revealEverything(call); err != nil {
			return err
		}
		rects, err := flattenForExport(call)
		if err != nil {
			return err
		}
		if slide < 1 || slide > len(rects) {
			return fmt.Errorf("เด็คนี้มี %d สไลด์ ไม่มีใบที่ %d", len(rects), slide)
		}
		shot, err = captureSlide(call, spec, rects[slide-1])
		return err
	})
	if err != nil {
		return nil, err
	}
	return shot, nil
}
