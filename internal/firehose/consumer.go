package firehose

type Consumer struct {
	WantedCollections map[string]struct{}
}

func (c *Consumer) wants(collection string) bool {
	if len(c.WantedCollections) == 0 {
		return true
	}
	_, ok := c.WantedCollections[collection]
	return ok
}
