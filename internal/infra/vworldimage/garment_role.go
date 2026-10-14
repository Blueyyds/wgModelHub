package vworldimage

// garmentRole is the VMind-Image product role contract, not a category or parser slot.
type garmentRole string

func (r garmentRole) valid() bool {
	switch r {
	case "outfit", "shoes", "bag", "accessory::ear", "accessory::head", "accessory::hair", "accessory::eye", "accessory::neck", "accessory::chest", "accessory::wrist", "accessory::hand", "accessory::finger", "belt", "clothing::top/full", "clothing::bottom", "accessory::other":
		return true
	default:
		return false
	}
}
