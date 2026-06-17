package provider

import "time"

// timeLayout is the canonical timestamp format the provider stores in state.
// FerrisKey returns RFC3339 date-times; normalising to RFC3339Nano keeps the
// rendering stable across reads.
const timeLayout = time.RFC3339Nano
