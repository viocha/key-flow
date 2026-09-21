package main

func isSupportedEdgeZone(zone string) bool {
	switch zone {
	case "topLeft", "top", "topRight", "left", "right", "bottomLeft", "bottom", "bottomRight":
		return true
	default:
		return false
	}
}
