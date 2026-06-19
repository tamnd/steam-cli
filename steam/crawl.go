package steam

import (
	"context"
	"fmt"
)

// crawl.go walks the public graph breadth-first. Starting from one seed (an app, a
// package, or a profile), it follows the typed edges the records already carry: an
// app reaches its DLC, demos, base game, and packages; a package reaches the apps
// it bundles; a profile reaches its most-played apps. It emits one CrawlNode per
// visited record, to a depth and a total node limit, so a reader can map the
// estate around a starting point without writing the traversal by hand. The walk
// never revisits a node and is best-effort past the seed: a node that cannot be
// fetched (an unreachable community host, a delisted app) is skipped, not fatal.

type crawlTarget struct {
	kind   string
	id     string
	depth  int
	parent string
}

type neighbor struct {
	kind string
	id   string
}

// Crawl walks the graph from seed, emitting each visited node. maxDepth bounds how
// far from the seed the walk goes (0 visits only the seed); limit bounds the total
// number of nodes emitted.
func (c *Client) Crawl(ctx context.Context, seed string, maxDepth, limit int, emit func(*CrawlNode) error) error {
	r := Classify(seed)
	kind := r.Kind
	if kind == "vanity" {
		kind = "profile"
	}
	if kind != "app" && kind != "package" && kind != "profile" {
		return fmt.Errorf("%w: crawl starts at an app, package, or profile, not %q", ErrUsage, r.Kind)
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxDepth < 0 {
		maxDepth = 0
	}

	queue := []crawlTarget{{kind: kind, id: r.ID, depth: 0}}
	visited := map[string]bool{}
	emitted := 0
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		key := t.kind + ":" + t.id
		if visited[key] {
			continue
		}
		visited[key] = true

		node, neighbors, err := c.fetchNode(ctx, t)
		if err != nil {
			if t.parent == "" {
				return mapErr(err) // the seed must resolve
			}
			continue // skip an unreachable node, keep walking the rest
		}
		if err := emit(node); err != nil {
			return err
		}
		emitted++
		if emitted >= limit {
			return nil
		}
		if t.depth >= maxDepth {
			continue
		}
		for _, nb := range neighbors {
			if !visited[nb.kind+":"+nb.id] {
				queue = append(queue, crawlTarget{kind: nb.kind, id: nb.id, depth: t.depth + 1, parent: key})
			}
		}
	}
	return nil
}

// fetchNode fetches the record for one target and returns it as a CrawlNode plus
// the neighbors to enqueue.
func (c *Client) fetchNode(ctx context.Context, t crawlTarget) (*CrawlNode, []neighbor, error) {
	node := &CrawlNode{
		ID:     t.kind + ":" + t.id,
		Kind:   t.kind,
		Ref:    t.id,
		Depth:  t.depth,
		Parent: t.parent,
		URL:    URLFor(t.kind, t.id),
	}
	var nbs []neighbor
	switch t.kind {
	case "app":
		a, err := c.appCore(ctx, t.id)
		if err != nil {
			return nil, nil, err
		}
		node.Name = a.Name
		node.URL = a.URL
		nbs = appNeighbors(a)
	case "package":
		p, err := c.Package(ctx, t.id)
		if err != nil {
			return nil, nil, err
		}
		node.Name = p.Name
		node.URL = p.URL
		for _, id := range p.AppRefs {
			nbs = append(nbs, neighbor{kind: "app", id: id})
		}
	case "profile":
		pr, err := c.Profile(ctx, t.id)
		if err != nil {
			return nil, nil, err
		}
		node.Name = pr.PersonaName
		node.URL = pr.URL
		for _, id := range pr.MostPlayedRefs {
			nbs = append(nbs, neighbor{kind: "app", id: id})
		}
	default:
		return nil, nil, ErrNotFound
	}
	for _, nb := range nbs {
		node.Edges = append(node.Edges, nb.kind+":"+nb.id)
	}
	return node, nbs, nil
}

// appNeighbors gathers an app's walkable neighbors: its DLC, demos, base game, and
// packages. Reviews and news point back at the same app, so they are not edges.
func appNeighbors(a *App) []neighbor {
	var nbs []neighbor
	for _, id := range a.DLCRefs {
		nbs = append(nbs, neighbor{kind: "app", id: id})
	}
	for _, id := range a.DemoRefs {
		nbs = append(nbs, neighbor{kind: "app", id: id})
	}
	if a.FullgameRef != "" {
		nbs = append(nbs, neighbor{kind: "app", id: a.FullgameRef})
	}
	for _, id := range a.PackageRefs {
		nbs = append(nbs, neighbor{kind: "package", id: id})
	}
	return nbs
}
