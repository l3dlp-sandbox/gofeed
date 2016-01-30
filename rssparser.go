package feed

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mmcdole/go-xpp"
)

type RSSParser struct {
	// Map of all namespaces (url / prefix)
	// that have been defined in the feed.
	feedSpaces map[string]string
}

func (rp *RSSParser) ParseFeed(feed string) (rss *RSSFeed, err error) {
	rp.feedSpaces = map[string]string{}
	p := xpp.NewXMLPullParser(strings.NewReader(feed))

	_, err = p.NextTag()
	if err != nil {
		return
	}

	return rp.parseRoot(p)
}

func (rp *RSSParser) parseRoot(p *xpp.XMLPullParser) (rss *RSSFeed, err error) {
	rssErr := p.Expect(xpp.StartTag, "rss")
	rdfErr := p.Expect(xpp.StartTag, "RDF")
	if rssErr != nil && rdfErr != nil {
		return nil, fmt.Errorf("%s or %s", rssErr.Error(), rdfErr.Error())
	}

	items := []*RSSItem{}
	ver := rp.parseVersion(p)

	for {
		tok, err := p.NextTag()
		if err != nil {
			return nil, err
		}

		if tok == xpp.EndTag {
			break
		}

		if tok == xpp.StartTag {
			if p.Name == "channel" {
				rss, err = rp.parseChannel(p)
				if err != nil {
					return nil, err
				}
			} else if p.Name == "item" {
				// Earlier versions of the RSS spec had "item" elements at the same
				// root level as "channel" elements.  We will merge these items
				// with any channel level items.
				item, err := rp.parseItem(p)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			} else {
				p.Skip()
			}
		}
	}

	rssErr = p.Expect(xpp.EndTag, "rss")
	rdfErr = p.Expect(xpp.EndTag, "RDF")
	if rssErr != nil && rdfErr != nil {
		return nil, fmt.Errorf("%s or %s", rssErr.Error(), rdfErr.Error())
	}

	if rss == nil {
		return nil, errors.New("No channel element found.")
	}

	rss.Items = append(rss.Items, items...)
	rss.Version = ver
	return rss, nil
}

func (rp *RSSParser) parseChannel(p *xpp.XMLPullParser) (rss *RSSFeed, err error) {

	if err = p.Expect(xpp.StartTag, "channel"); err != nil {
		return nil, err
	}

	rss = &RSSFeed{}
	rss.Items = []*RSSItem{}
	rss.Categories = []RSSCategory{}
	rss.Extensions = map[string]map[string][]Extension{}

	for {
		tok, err := p.NextTag()
		if err != nil {
			return nil, err
		}

		if tok == xpp.EndTag {
			break
		}

		if tok == xpp.StartTag {

			// Parse and store any namespace prefix/url attributes
			rp.parseNamespaces(p)

			if p.Name == "title" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Title = result
			} else if p.Name == "description" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Description = result
			} else if p.Name == "link" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Link = result
			} else if p.Name == "language" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Language = result
			} else if p.Name == "copyright" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Copyright = result
			} else if p.Name == "managingEditor" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.ManagingEditor = result
			} else if p.Name == "webMaster" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.WebMaster = result
			} else if p.Name == "pubDate" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.PubDate = result
				date, err := ParseDate(result)
				if err == nil {
					rss.PubDateParsed = date
				}
			} else if p.Name == "lastBuildDate" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.LastBuildDate = result
				date, err := ParseDate(result)
				if err == nil {
					rss.PubDateParsed = date
				}
			} else if p.Name == "generator" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Generator = result
			} else if p.Name == "docs" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Docs = result
			} else if p.Name == "ttl" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.TTL = result
			} else if p.Name == "rating" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				rss.Rating = result
			} else if p.Name == "item" {
				result, err := rp.parseItem(p)
				if err != nil {
					return nil, err
				}
				rss.Items = append(rss.Items, result)
			} else if p.Name == "category" {
				result, err := rp.parseCategory(p)
				if err != nil {
					return nil, err
				}
				rss.Categories = append(rss.Categories, result)
			} else if p.Space != "" {
				result, err := rp.parseExtension(p)
				if err != nil {
					return nil, err
				}
				prefix := rp.prefixForNamespace(p.Space)

				// Ensure the extension prefix map exists
				if _, ok := rss.Extensions[prefix]; !ok {
					rss.Extensions[prefix] = map[string][]Extension{}
				}
				// Ensure the extension element slice exists
				if _, ok := rss.Extensions[prefix][p.Name]; !ok {
					rss.Extensions[prefix][p.Name] = []Extension{}
				}

				rss.Extensions[prefix][p.Name] = append(rss.Extensions[prefix][p.Name], result)
			} else {
				// Skip element as it isn't an extension and not
				// part of the spec
				p.Skip()
			}
		}
	}

	if err = p.Expect(xpp.EndTag, "channel"); err != nil {
		return nil, err
	}

	return rss, nil
}

func (rp *RSSParser) parseItem(p *xpp.XMLPullParser) (item *RSSItem, err error) {
	if err = p.Expect(xpp.StartTag, "item"); err != nil {
		return nil, err
	}

	item = &RSSItem{}
	item.Categories = []RSSCategory{}

	for {
		tok, err := p.NextTag()
		if err != nil {
			return nil, err
		}

		if tok == xpp.EndTag {
			break
		}

		if tok == xpp.StartTag {

			// Parse and store any namespace prefix/url attributes
			rp.parseNamespaces(p)

			if p.Name == "title" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.Title = result
			} else if p.Name == "description" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.Description = result
			} else if p.Name == "link" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.Link = result
			} else if p.Name == "author" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.Author = result
			} else if p.Name == "comments" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.Comments = result
			} else if p.Name == "pubDate" {
				result, err := p.NextText()
				if err != nil {
					return nil, err
				}
				item.PubDate = result
				date, err := ParseDate(result)
				if err == nil {
					item.PubDateParsed = date
				}
			} else if p.Name == "source" {
				result, err := rp.parseSource(p)
				if err != nil {
					return nil, err
				}
				item.Source = result
			} else if p.Name == "enclosure" {
				result, err := rp.parseEnclosure(p)
				if err != nil {
					return nil, err
				}
				item.Enclosure = result
			} else if p.Name == "guid" {
				result, err := rp.parseGuid(p)
				if err != nil {
					return nil, err
				}
				item.Guid = result
			} else if p.Name == "category" {
				result, err := rp.parseCategory(p)
				if err != nil {
					return nil, err
				}
				item.Categories = append(item.Categories, result)
			} else {
				// Skip any elements not part of the item spec
				p.Skip()
			}
		}
	}

	if err = p.Expect(xpp.EndTag, "item"); err != nil {
		return nil, err
	}

	return item, nil
}

func (rp *RSSParser) parseSource(p *xpp.XMLPullParser) (source RSSSource, err error) {
	if err = p.Expect(xpp.StartTag, "source"); err != nil {
		return source, err
	}

	source.URL = p.Attribute("url")

	result, err := p.NextText()
	if err != nil {
		return source, err
	}
	source.Title = result

	if err = p.Expect(xpp.EndTag, "source"); err != nil {
		return source, err
	}
	return source, nil
}

func (rp *RSSParser) parseEnclosure(p *xpp.XMLPullParser) (enclosure RSSEnclosure, err error) {
	if err = p.Expect(xpp.StartTag, "enclosure"); err != nil {
		return enclosure, err
	}

	enclosure.URL = p.Attribute("url")
	enclosure.Length = p.Attribute("length")
	enclosure.Type = p.Attribute("type")

	if err = p.Expect(xpp.EndTag, "enclosure"); err != nil {
		return enclosure, err
	}

	return enclosure, nil
}

func (rp *RSSParser) parseImage(p *xpp.XMLPullParser) (image RSSImage, err error) {
	if err = p.Expect(xpp.StartTag, "image"); err != nil {
		return image, err
	}

	for {
		tok, err := p.NextTag()
		if err != nil {
			return image, err
		}

		if tok == xpp.EndTag {
			break
		}

		if tok == xpp.StartTag {
			if p.Name == "url" {
				result, err := p.NextText()
				if err != nil {
					return image, err
				}
				image.URL = result
			} else if p.Name == "title" {
				result, err := p.NextText()
				if err != nil {
					return image, err
				}
				image.Title = result
			} else if p.Name == "link" {
				result, err := p.NextText()
				if err != nil {
					return image, err
				}
				image.Link = result
			} else if p.Name == "width" {
				result, err := p.NextText()
				if err != nil {
					return image, err
				}
				image.Width = result
			} else if p.Name == "height" {
				result, err := p.NextText()
				if err != nil {
					return image, err
				}
				image.Height = result
			} else {
				p.Skip()
			}
		}
	}

	if err = p.Expect(xpp.EndTag, "image"); err != nil {
		return image, err
	}

	return image, nil
}

func (rp *RSSParser) parseGuid(p *xpp.XMLPullParser) (guid RSSGuid, err error) {
	if err = p.Expect(xpp.StartTag, "guid"); err != nil {
		return guid, err
	}

	guid.IsPermalink = p.Attribute("isPermalink")

	result, err := p.NextText()
	if err != nil {
		return
	}
	guid.Value = result

	if err = p.Expect(xpp.EndTag, "guid"); err != nil {
		return guid, err
	}

	return guid, nil
}

func (rp *RSSParser) parseCategory(p *xpp.XMLPullParser) (cat RSSCategory, err error) {

	if err = p.Expect(xpp.StartTag, "category"); err != nil {
		return cat, err
	}

	cat.Domain = p.Attribute("domain")

	result, err := p.NextText()
	if err != nil {
		return cat, err
	}

	cat.Value = result

	if err = p.Expect(xpp.EndTag, "category"); err != nil {
		return cat, err
	}
	return cat, nil
}

func (rp *RSSParser) parseExtension(p *xpp.XMLPullParser) (ext Extension, err error) {
	if err = p.Expect(xpp.StartTag, "*"); err != nil {
		return ext, err
	}

	ext.Name = p.Name
	ext.Attrs = map[string]string{}
	ext.Children = map[string][]Extension{}

	for _, attr := range p.Attrs {
		// TODO: Alright that we are stripping
		// namespace information from attributes
		// for the usecase of feed parsing?
		ext.Attrs[attr.Name.Local] = attr.Value
	}

	for {
		tok, err := p.Next()
		if err != nil {
			return ext, err
		}

		if tok == xpp.EndTag {
			break
		}

		if tok == xpp.StartTag {
			child, err := rp.parseExtension(p)
			if err != nil {
				return ext, err
			}

			if _, ok := ext.Children[child.Name]; !ok {
				ext.Children[child.Name] = []Extension{}
			}

			ext.Children[child.Name] = append(ext.Children[child.Name], child)
		} else if tok == xpp.Text {
			ext.Value = p.Text
		}
	}

	if err = p.Expect(xpp.EndTag, ext.Name); err != nil {
		return ext, err
	}

	return ext, nil
}

func (rp *RSSParser) prefixForNamespace(space string) string {
	lspace := strings.ToLower(space)

	// First we check if the global namespace map
	// contains an entry for this namespace/prefix.
	// This way we can use the canonical prefix for this
	// ns instead of the one defined in the feed.
	if prefix, ok := globalNamespaces[lspace]; ok {
		return prefix
	}

	// Next we check if the feed itself defined this
	// this namespace and return it if we have a result.
	if prefix, ok := rp.feedSpaces[lspace]; ok {
		return prefix
	}

	// Lastly, any namespace which is not defined in the
	// the feed will be the prefix itself when using Go's
	// xml.Decoder.Token() method.
	return space
}

func (rp *RSSParser) parseNamespaces(p *xpp.XMLPullParser) {
	for _, attr := range p.Attrs {
		if attr.Name.Space == "xmlns" {
			spacePrefix := strings.ToLower(attr.Name.Local)
			rp.feedSpaces[attr.Value] = spacePrefix
		}
	}
}

func (rp *RSSParser) parseVersion(p *xpp.XMLPullParser) (ver string) {
	if p.Name == "rss" {
		ver = p.Attribute("version")
		if ver == "" {
			ver = "2.0"
		}
	} else if p.Name == "RDF" {
		ns := p.Attribute("xmlns")
		if ns == "http://channel.netscape.com/rdf/simple/0.9/" {
			ver = "0.9"
		} else if ns == "http://purl.org/rss/1.0/" {
			ver = "1.0"
		}
	}
	return
}