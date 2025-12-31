package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderHeader(t *testing.T) {
	tabs := []Tab{
		NewPlaceholderTab(TabVMs, "VMs", "1", ""),
		NewPlaceholderTab(TabCreate, "Create", "2", ""),
	}

	header := renderHeader(tabs, 0, 100)

	// Should contain title
	assert.Contains(t, header, "ucli")
	assert.Contains(t, header, "VM Manager")

	// Should contain tab names
	assert.Contains(t, header, "VMs")
	assert.Contains(t, header, "Create")

	// Should contain quit hint (could be "[q]uit" or "quit")
	assert.True(t, strings.Contains(header, "quit") || strings.Contains(header, "[q]uit"))
}

func TestRenderHeader_ActiveTab(t *testing.T) {
	tabs := []Tab{
		NewPlaceholderTab(TabVMs, "VMs", "1", ""),
		NewPlaceholderTab(TabCreate, "Create", "2", ""),
	}

	// Active tab 0
	header := renderHeader(tabs, 0, 100)
	assert.Contains(t, header, "VMs")

	// Active tab 1
	header = renderHeader(tabs, 1, 100)
	assert.Contains(t, header, "Create")
}

func TestRenderHeader_NoTabs(t *testing.T) {
	header := renderHeader(nil, 0, 100)

	// Should still contain title
	assert.Contains(t, header, "ucli")
}

func TestRenderTabBar(t *testing.T) {
	tabs := []TabInfo{
		{Name: "VMs", ShortKey: "1", Active: true},
		{Name: "Create", ShortKey: "2", Active: false},
	}

	tabBar := RenderTabBar(tabs)

	assert.Contains(t, tabBar, "VMs")
	assert.Contains(t, tabBar, "Create")
	assert.Contains(t, tabBar, "[1]")
	assert.Contains(t, tabBar, "[2]")
}

func TestRenderTabBar_Empty(t *testing.T) {
	tabBar := RenderTabBar(nil)
	assert.Equal(t, "", tabBar)
}

func TestRenderTabBar_SingleTab(t *testing.T) {
	tabs := []TabInfo{
		{Name: "VMs", ShortKey: "1", Active: true},
	}

	tabBar := RenderTabBar(tabs)

	assert.Contains(t, tabBar, "VMs")
	assert.Contains(t, tabBar, "[1]")
}
