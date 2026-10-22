// Package similar scores how likely two work items describe the same work.
//
// The score is deterministic and dependency-free: character trigrams over a
// normalized title catch typos and inflection ("авторизация"/"авторизации"),
// word stems catch reordered titles, and the description adds weight only when
// both sides have one. It does not understand paraphrases with different
// vocabulary; that is an explicit limitation, not a hidden promise.
package similar

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Doc is the comparable fingerprint of one work item.
type Doc struct {
	title    map[string]struct{}
	words    map[string]struct{}
	desc     map[string]struct{}
	hasDesc  bool
	numbers  map[string]struct{}
	Stems    []string
	External string
}

// maxDescription bounds work per item; long descriptions keep their opening.
const maxDescription = 2000

var stop = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`и в во не что он на я с со как а то все она так его но да ты к у же вы за бы по только ее мне было вот от меня еще нет о из ему теперь когда даже ну вдруг ли если уже или ни быть был него до вас нибудь опять уж вам ведь там потом себя ничего ей может они тут где есть надо ней для мы тебя их чем была сам чтоб без будто чего раз тоже себе под будет ж тогда кто этот того потому этого какой совсем ним здесь этом один почти мой тем чтобы нее сейчас были куда зачем всех никогда можно при наконец два об другой хоть после над больше тот через эти нас про всего них какая много разве три эту моя впрочем хорошо свою этой перед иногда лучше чуть том нельзя такой им более всегда конечно всю между нужно сделать добавить сделай исправить нужно нужен нужна реализовать
the a an and or of to in on for with by at from is are be this that it as into via add fix make update implement new`) {
		stop[w] = true
	}
}

// New builds a fingerprint. Titles are bounded by the caller (≤200 runes).
func New(title, description, external string) Doc {
	d := Doc{title: map[string]struct{}{}, words: map[string]struct{}{}, desc: map[string]struct{}{}, External: strings.TrimSpace(external)}
	d.numbers = map[string]struct{}{}
	tw := words(title)
	for _, w := range tw {
		d.words[canonical(w)] = struct{}{}
		if strings.IndexFunc(w, unicode.IsDigit) >= 0 {
			d.numbers[w] = struct{}{}
		}
	}
	addTrigrams(d.title, tw)
	if utf8.RuneCountInString(description) > maxDescription {
		description = string([]rune(description)[:maxDescription])
	}
	dw := words(description)
	if len(dw) >= 3 {
		d.hasDesc = true
		addTrigrams(d.desc, dw)
	}
	for w := range d.words {
		d.Stems = append(d.Stems, w)
	}
	sort.Strings(d.Stems)
	return d
}

// Score returns 0..1 and the strongest signals that produced it.
func Score(a, b Doc) (float64, []string) {
	if a.External != "" && a.External == b.External {
		return 1, []string{"external_id"}
	}
	title := dice(a.title, b.title)
	if w := jaccard(a.words, b.words); w > title {
		title = w
	}
	// "Ревью T-12" and "Ревью T-13" share wording but name different targets.
	if len(a.numbers) > 0 && len(b.numbers) > 0 && jaccard(a.numbers, b.numbers) == 0 {
		title *= 0.5
	}
	score, reasons := title, []string{"title"}
	if a.hasDesc && b.hasDesc {
		desc := dice(a.desc, b.desc)
		if mixed := 0.6*title + 0.4*desc; mixed > score {
			score = mixed
		}
		if desc >= 0.5 {
			reasons = append(reasons, "description")
		}
	}
	return round(score), reasons
}

func words(s string) []string {
	s = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "ё", "е"), "Ё", "е"))
	out := []string{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if !stop[w] && utf8.RuneCountInString(w) > 1 {
			out = append(out, w)
		}
	}
	return out
}

// synonyms map common RU/EN development vocabulary to one concept, so
// "Вход через GitHub" and "GitHub OAuth login" share their key terms. Longer
// prefixes are checked first; the list is deliberately small and explicit.
var synonyms = [][2]string{
	{"входящ", "incoming"}, {"автори", "auth"}, {"аутент", "auth"}, {"логин", "auth"}, {"login", "auth"}, {"signin", "auth"}, {"oauth", "auth"}, {"auth", "auth"}, {"вход", "auth"},
	{"выход", "logout"}, {"разлог", "logout"}, {"logout", "logout"}, {"signout", "logout"},
	{"экспорт", "export"}, {"выгруз", "export"}, {"export", "export"}, {"импорт", "import"}, {"import", "import"},
	{"темн", "dark"}, {"dark", "dark"}, {"тема", "theme"}, {"темы", "theme"}, {"теме", "theme"}, {"тему", "theme"}, {"темой", "theme"}, {"theme", "theme"},
	{"тест", "test"}, {"test", "test"}, {"ошибк", "bug"}, {"баг", "bug"}, {"bug", "bug"}, {"падени", "bug"}, {"падают", "bug"}, {"падает", "bug"}, {"crash", "bug"}, {"краш", "bug"},
	{"экран", "screen"}, {"страниц", "screen"}, {"page", "screen"}, {"screen", "screen"},
	{"настрой", "settings"}, {"settings", "settings"}, {"конфиг", "settings"}, {"config", "settings"},
	{"пользов", "user"}, {"юзер", "user"}, {"user", "user"}, {"кнопк", "button"}, {"button", "button"},
	{"пагина", "pagination"}, {"paginat", "pagination"}, {"поиск", "search"}, {"search", "search"},
	{"список", "list"}, {"списк", "list"}, {"list", "list"}, {"задач", "task"}, {"task", "task"},
	{"панел", "panel"}, {"panel", "panel"}, {"дашборд", "panel"}, {"dashboard", "panel"},
	{"агент", "agent"}, {"agent", "agent"}, {"уведом", "notify"}, {"notif", "notify"},
	{"миграц", "migrate"}, {"migrat", "migrate"}, {"database", "db"}, {"sqlite", "db"}, {"бд", "db"},
	{"ревью", "review"}, {"review", "review"}, {"документ", "docs"}, {"docs", "docs"}, {"readme", "docs"},
}

func canonical(w string) string {
	for _, pair := range synonyms {
		if strings.HasPrefix(w, pair[0]) {
			return pair[1]
		}
	}
	return stem(w)
}

// stem keeps a short prefix: crude, but stable for Russian inflection and
// English plurals without a language dictionary.
func stem(w string) string {
	r := []rune(w)
	if len(r) > 6 {
		r = r[:6]
	}
	return string(r)
}

func addTrigrams(set map[string]struct{}, words []string) {
	for _, w := range words {
		r := []rune(" " + w + " ")
		for i := 0; i+3 <= len(r); i++ {
			set[string(r[i:i+3])] = struct{}{}
		}
	}
}

func dice(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	shared := 0
	for k := range a {
		if _, ok := b[k]; ok {
			shared++
		}
	}
	return 2 * float64(shared) / float64(len(a)+len(b))
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := 0
	for k := range a {
		if _, ok := b[k]; ok {
			shared++
		}
	}
	return float64(shared) / float64(len(a)+len(b)-shared)
}

func round(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
