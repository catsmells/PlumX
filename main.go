package main
import (
	"fmt"
	"image/color"
	_ "image/gif"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)
type brightTheme struct{}
var (
	colBackground = color.NRGBA{R: 0xf8, G: 0xf6, B: 0xfc, A: 0xff}
	colForeground = color.NRGBA{R: 0x2a, G: 0x22, B: 0x30, A: 0xff}
	colPrimary    = color.NRGBA{R: 0x7c, G: 0x4d, B: 0xa6, A: 0xff}
	colHover      = color.NRGBA{R: 0xec, G: 0xe3, B: 0xf7, A: 0xff}
	colSelection  = color.NRGBA{R: 0xdc, G: 0xc9, B: 0xf2, A: 0xff}
	colCard       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	colCardBorder = color.NRGBA{R: 0xe4, G: 0xdc, B: 0xef, A: 0xff}
	colMuted      = color.NRGBA{R: 0x8a, G: 0x7d, B: 0x96, A: 0xff}
)
func (brightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameForeground:
		return colForeground
	case theme.ColorNamePrimary:
		return colPrimary
	case theme.ColorNameHover:
		return colHover
	case theme.ColorNameSelection:
		return colSelection
	case theme.ColorNameInputBackground:
		return colCard
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return colCard
	case theme.ColorNameSeparator:
		return colCardBorder
	case theme.ColorNameDisabled:
		return colMuted
	}
	return theme.DefaultTheme().Color(name, variant)
}
func (brightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
func (brightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}
func (brightTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInlineIcon:
		return 22
	case theme.SizeNameText:
		return 13
	}
	return theme.DefaultTheme().Size(name)
}
type fileEntry struct {
	name  string
	path  string
	isDir bool
	kind  string
	size  int64
}
type fileCell struct {
	widget.BaseWidget
	content     fyne.CanvasObject
	path        string
	isDir       bool
	onTap       func()
	onSecondary func(*fyne.PointEvent)
	onDragged   func(*fyne.DragEvent)
	onDragEnd   func()
}
func newFileCell(icon fyne.Resource, name string, onTap func(), onSecondary func(*fyne.PointEvent)) *fileCell {
	label := widget.NewLabel(name)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	c := &fileCell{
		content:     container.NewVBox(container.NewCenter(widget.NewIcon(icon)), label),
		onTap:       onTap,
		onSecondary: onSecondary,
	}
	c.ExtendBaseWidget(c)
	return c
}
func (c *fileCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}
func newFileRow(icon fyne.Resource, f fileEntry, onTap func(), onSecondary func(*fyne.PointEvent)) *fileCell {
	nameLabel := widget.NewLabel(f.name)
	nameLabel.Truncation = fyne.TextTruncateEllipsis
	kindText := strings.ToUpper(f.kind[:1]) + f.kind[1:]
	if f.isDir {
		kindText = "Folder"
	}
	kindLabel := widget.NewLabel(kindText)
	sizeText := ""
	if !f.isDir {
		sizeText = formatSize(f.size)
	}
	sizeLabel := widget.NewLabel(sizeText)
	sizeLabel.Alignment = fyne.TextAlignTrailing
	left := container.NewBorder(nil, nil, widget.NewIcon(icon), nil, nameLabel)
	right := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(70, kindLabel.MinSize().Height), kindLabel),
		container.NewGridWrap(fyne.NewSize(70, sizeLabel.MinSize().Height), sizeLabel),
	)
	row := container.NewBorder(nil, nil, nil, right, left)
	c := &fileCell{content: row, onTap: onTap, onSecondary: onSecondary}
	c.ExtendBaseWidget(c)
	return c
}
func (c *fileCell) TappedSecondary(e *fyne.PointEvent) {
	if c.onSecondary != nil {
		c.onSecondary(e)
	}
}
type secondaryButton struct {
	*widget.Button
	onSecondary func(*fyne.PointEvent)
}
func (b *secondaryButton) TappedSecondary(e *fyne.PointEvent) {
	if b.onSecondary != nil {
		b.onSecondary(e)
	}
}
func (c *fileCell) Tapped(_ *fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}
func (c *fileCell) Dragged(e *fyne.DragEvent) {
	if c.onDragged != nil {
		c.onDragged(e)
	}
}
func (c *fileCell) DragEnd() {
	if c.onDragEnd != nil {
		c.onDragEnd()
	}
}
func card(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colCard)
	bg.CornerRadius = 10
	bg.StrokeColor = colCardBorder
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(content))
}
func toolButton(icon fyne.Resource, tapped func()) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.NewPrimaryThemedResource(icon), tapped)
	b.Importance = widget.LowImportance
	return b
}
func iconFor(kind string) fyne.Resource {
	switch kind {
	case "dir":
		return theme.FolderIcon()
	case "image":
		return theme.FileImageIcon()
	case "audio":
		return theme.MediaMusicIcon()
	case "doc":
		return theme.DocumentIcon()
	default:
		return theme.FileIcon()
	}
}
func kindForFile(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return "image"
	case ".mp3", ".flac", ".wav", ".ogg", ".m4a", ".aac":
		return "audio"
	case ".pdf", ".doc", ".docx", ".odt", ".txt", ".md":
		return "doc"
	default:
		return "file"
	}
}
func fileEntryFor(path string, e os.DirEntry) fileEntry {
	fe := fileEntry{name: e.Name(), path: path, isDir: e.IsDir()}
	if fe.isDir {
		fe.kind = "dir"
	} else {
		fe.kind = kindForFile(fe.name)
		if info, err := e.Info(); err == nil {
			fe.size = info.Size()
		}
	}
	return fe
}
func listDir(dir string, showHidden bool) ([]fileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		if !showHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		files = append(files, fileEntryFor(filepath.Join(dir, e.Name()), e))
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].isDir != files[j].isDir {
			return files[i].isDir
		}
		return strings.ToLower(files[i].name) < strings.ToLower(files[j].name)
	})
	return files, nil
}
func formatSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(n)
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds + 0.5)
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}
func statusText(files []fileEntry) string {
	var totalSize int64
	for _, f := range files {
		if !f.isDir {
			totalSize += f.size
		}
	}
	return fmt.Sprintf("%d items · %s", len(files), formatSize(totalSize))
}
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
func main() {
	a := app.New()
	a.Settings().SetTheme(brightTheme{})
	w := a.NewWindow("PlumX")
	w.Resize(fyne.NewSize(1000, 640))
	var (
		currentPath string
		backStack   []string
		fwdStack    []string
		viewMode    = "grid"
	)
	var navigateTo func(path string)
	var goBack, goForward, goUp, doRefresh func()
	var loadPath func(path string) error
	var selectedPath string
	var selectFile func(f fileEntry)
	var player *mpvPlayer
	var showFileContextMenu func(f fileEntry, e *fyne.PointEvent)
	showHiddenCheck := widget.NewCheck("Hidden", nil)
	var sidebarTargets, folderTargets []dropTarget
	var displayedFiles []fileEntry
	var renderFiles func(files []fileEntry)
	status := widget.NewLabel("")
	status.TextStyle = fyne.TextStyle{Bold: true}
	status.Importance = widget.HighImportance
	addressBar := widget.NewEntry()
	addressBar.OnSubmitted = func(path string) { navigateTo(path) }
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search files and folders…")
	limitCheck := widget.NewCheck("This folder", nil)
	searchBox := container.NewBorder(nil, nil, nil, limitCheck, searchEntry)
	searchBox.Hide()
	var searchGen atomic.Int64
	startSearch := func(query string) {
		query = strings.TrimSpace(query)
		gen := searchGen.Add(1)
		if query == "" {
			status.SetText(statusText(displayedFiles))
			return
		}
		root := "/"
		if limitCheck.Checked {
			root = currentPath
		}
		showHidden := showHiddenCheck.Checked
		status.SetText(fmt.Sprintf("Searching %s for %q…", root, query))
		const maxResults = 2000
		var mu sync.Mutex
		var results []fileEntry
		capped := false
		pushUpdate := func(done bool) {
			mu.Lock()
			snapshot := append([]fileEntry{}, results...)
			isCapped := capped
			mu.Unlock()
			fyne.Do(func() {
				if searchGen.Load() != gen {
					return
				}
				displayedFiles = snapshot
				renderFiles(snapshot)
				switch {
				case done && isCapped:
					status.SetText(fmt.Sprintf("Showing first %d matches for %q in %s (more exist)", len(snapshot), query, root))
				case done:
					status.SetText(fmt.Sprintf("%d match(es) for %q in %s", len(snapshot), query, root))
				default:
					status.SetText(fmt.Sprintf("Searching… %d match(es) so far", len(snapshot)))
				}
			})
		}
		go func() {
			lastPush := time.Now()
			searchFiles(root, query, showHidden, func() bool {
				return searchGen.Load() != gen || capped
			}, func(f fileEntry) {
				mu.Lock()
				results = append(results, f)
				n := len(results)
				if n >= maxResults {
					capped = true
				}
				mu.Unlock()
				if capped || time.Since(lastPush) > 150*time.Millisecond {
					pushUpdate(false)
					lastPush = time.Now()
				}
			})
			if searchGen.Load() == gen {
				pushUpdate(true)
			}
		}()
	}
	searchEntry.OnSubmitted = startSearch
	setSearchActive := func(active bool) {
		if active {
			addressBar.Hide()
			searchBox.Show()
			w.Canvas().Focus(searchEntry)
		} else {
			searchGen.Add(1)
			searchEntry.SetText("")
			searchBox.Hide()
			addressBar.Show()
			loadPath(currentPath)
		}
	}
	navCard := card(container.NewHBox(
		toolButton(theme.NavigateBackIcon(), func() { goBack() }),
		toolButton(theme.NavigateNextIcon(), func() { goForward() }),
		toolButton(theme.MoveUpIcon(), func() { goUp() }),
		toolButton(theme.ViewRefreshIcon(), func() { doRefresh() }),
	))
	var searchActive bool
	showHiddenCheck.OnChanged = func(bool) { loadPath(currentPath) }
	viewCard := card(container.NewHBox(
		toolButton(theme.GridIcon(), func() {
			viewMode = "grid"
			renderFiles(displayedFiles)
		}),
		toolButton(theme.ListIcon(), func() {
			viewMode = "list"
			renderFiles(displayedFiles)
		}),
		toolButton(theme.SearchIcon(), func() {
			searchActive = !searchActive
			setSearchActive(searchActive)
		}),
		showHiddenCheck,
	))
	toolbar := container.NewBorder(nil, nil, navCard, viewCard, container.NewStack(addressBar, searchBox))
	grid := container.NewGridWrap(fyne.NewSize(110, 100))
	listBox := container.NewVBox()
	fileArea := container.NewVScroll(grid)
	var dragSourcePath string
	var dragHover dropTarget
	var dragHovering bool
	wireDrag := func(cell *fileCell, f fileEntry) {
		cell.path = f.path
		cell.isDir = f.isDir
		cell.onDragged = func(e *fyne.DragEvent) {
			if dragSourcePath == "" {
				dragSourcePath = f.path
			}
			all := append(append([]dropTarget{}, sidebarTargets...), folderTargets...)
			if t := hitTestTargets(e.AbsolutePosition, all, dragSourcePath); t != nil {
				dragHover, dragHovering = *t, true
				status.SetText(fmt.Sprintf("Move %q to %s", filepath.Base(dragSourcePath), filepath.Base(t.path)))
			} else {
				dragHovering = false
				status.SetText(fmt.Sprintf("Dragging %q…", filepath.Base(dragSourcePath)))
			}
		}
		cell.onDragEnd = func() {
			src, target, hadTarget := dragSourcePath, dragHover, dragHovering
			dragSourcePath, dragHovering = "", false
			if src == "" || !hadTarget {
				doRefresh()
				return
			}
			var err error
			if target.isTrash {
				err = moveToTrash(src)
			} else {
				err = moveFile(src, target.path)
			}
			doRefresh()
			if err != nil {
				status.SetText(fmt.Sprintf("Cannot move %s: %v", filepath.Base(src), err))
				return
			}
			status.SetText(fmt.Sprintf("Moved %s", filepath.Base(src)))
		}
	}
	renderFiles = func(files []fileEntry) {
		folderTargets = nil
		if viewMode == "list" {
			rows := make([]fyne.CanvasObject, 0, len(files))
			for _, f := range files {
				f := f
				row := newFileRow(iconFor(f.kind), f, func() {
					if f.isDir {
						navigateTo(f.path)
					} else {
						selectFile(f)
					}
				}, func(e *fyne.PointEvent) {
					showFileContextMenu(f, e)
				})
				wireDrag(row, f)
				if f.isDir {
					folderTargets = append(folderTargets, dropTarget{path: f.path, obj: row})
				}
				rows = append(rows, row)
			}
			listBox.Objects = rows
			listBox.Refresh()
			fileArea.Content = listBox
		} else {
			objs := make([]fyne.CanvasObject, 0, len(files))
			for _, f := range files {
				f := f
				cell := newFileCell(iconFor(f.kind), f.name, func() {
					if f.isDir {
						navigateTo(f.path)
					} else {
						selectFile(f)
					}
				}, func(e *fyne.PointEvent) {
					showFileContextMenu(f, e)
				})
				wireDrag(cell, f)
				if f.isDir {
					folderTargets = append(folderTargets, dropTarget{path: f.path, obj: cell})
				}
				objs = append(objs, cell)
			}
			grid.Objects = objs
			grid.Refresh()
			fileArea.Content = grid
		}
		fileArea.Refresh()
	}
	loadPath = func(path string) error {
		files, err := listDir(path, showHiddenCheck.Checked)
		if err != nil {
			return err
		}
		searchGen.Add(1)
		currentPath = path
		displayedFiles = files
		searchEntry.SetText("")
		renderFiles(files)
		addressBar.SetText(path)
		status.SetText(statusText(files))
		return nil
	}
	navigateTo = func(path string) {
		target, err := filepath.Abs(path)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		prev := currentPath
		if err := loadPath(target); err != nil {
			status.SetText(fmt.Sprintf("Cannot open %s: %v", target, err))
			return
		}
		if prev != "" && prev != target {
			backStack = append(backStack, prev)
			fwdStack = nil
		}
	}
	goBack = func() {
		if len(backStack) == 0 {
			return
		}
		target := backStack[len(backStack)-1]
		prev := currentPath
		if err := loadPath(target); err != nil {
			status.SetText(err.Error())
			return
		}
		backStack = backStack[:len(backStack)-1]
		fwdStack = append(fwdStack, prev)
	}
	goForward = func() {
		if len(fwdStack) == 0 {
			return
		}
		target := fwdStack[len(fwdStack)-1]
		prev := currentPath
		if err := loadPath(target); err != nil {
			status.SetText(err.Error())
			return
		}
		fwdStack = fwdStack[:len(fwdStack)-1]
		backStack = append(backStack, prev)
	}
	goUp = func() {
		if parent := filepath.Dir(currentPath); parent != currentPath {
			navigateTo(parent)
		}
	}
	doRefresh = func() {
		loadPath(currentPath)
	}
	w.SetOnDropped(func(_ fyne.Position, items []fyne.URI) {
		copied := 0
		var errs []string
		for _, item := range items {
			if item.Scheme() != "file" {
				continue
			}
			srcPath := item.Path()
			if _, err := os.Stat(srcPath); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			destPath := filepath.Join(currentPath, filepath.Base(srcPath))
			if err := copyAny(srcPath, destPath); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			copied++
		}
		doRefresh()
		if len(errs) > 0 {
			status.SetText(fmt.Sprintf("Copied %d item(s), %d error(s): %s", copied, len(errs), strings.Join(errs, "; ")))
		}
	})
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	places := []struct {
		label string
		icon  fyne.Resource
		path  string
	}{
		{"Home", theme.HomeIcon(), home},
		{"Desktop", theme.ComputerIcon(), filepath.Join(home, "Desktop")},
		{"Documents", theme.DocumentIcon(), filepath.Join(home, "Documents")},
		{"Downloads", theme.DownloadIcon(), filepath.Join(home, "Downloads")},
		{"Pictures", theme.FileImageIcon(), filepath.Join(home, "Pictures")},
		{"Music", theme.MediaMusicIcon(), filepath.Join(home, "Music")},
	}
	if trashPath, err := trashFilesDir(); err == nil {
		places = append(places, struct {
			label string
			icon  fyne.Resource
			path  string
		}{"Trash", theme.DeleteIcon(), trashPath})
	}
	sidebarItems := container.NewVBox()
	sidebarItems.Add(widget.NewLabelWithStyle("PLACES", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, p := range places {
		p := p
		b := widget.NewButtonWithIcon(p.label, p.icon, func() { navigateTo(p.path) })
		b.Alignment = widget.ButtonAlignLeading
		b.Importance = widget.LowImportance
		if p.label == "Trash" {
			sb := &secondaryButton{Button: b, onSecondary: func(e *fyne.PointEvent) {
				empty := fyne.NewMenuItem("Empty Trash", func() {
					if err := emptyTrash(); err != nil {
						status.SetText(fmt.Sprintf("Cannot empty trash: %v", err))
						return
					}
					doRefresh()
				})
				widget.NewPopUpMenu(fyne.NewMenu("", empty), w.Canvas()).ShowAtPosition(e.AbsolutePosition)
			}}
			sb.ExtendBaseWidget(sb)
			sidebarItems.Add(sb)
			sidebarTargets = append(sidebarTargets, dropTarget{path: p.path, obj: sb, isTrash: true})
			continue
		}
		sidebarItems.Add(b)
		sidebarTargets = append(sidebarTargets, dropTarget{path: p.path, obj: b})
	}
	sidebarItems.Add(widget.NewSeparator())
	sidebarItems.Add(widget.NewLabelWithStyle("DEVICES", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	type device struct {
		name      string
		icon      fyne.Resource
		usedFrac  float64
		freeLabel string
	}
	devices := []device{
		{"Filesystem", theme.StorageIcon(), 0.75, "118 GB free of 476 GB"},
		{"USB Drive", theme.StorageIcon(), 0.37, "20 GB free of 32 GB"},
	}
	for _, d := range devices {
		nameRow := container.NewHBox(
			widget.NewIcon(theme.NewPrimaryThemedResource(d.icon)),
			widget.NewLabel(d.name),
		)
		bar := widget.NewProgressBar()
		bar.Value = d.usedFrac
		bar.TextFormatter = func() string { return "" }
		freeLabel := widget.NewLabel(d.freeLabel)
		freeLabel.TextStyle = fyne.TextStyle{Italic: true}
		sidebarItems.Add(card(container.NewVBox(nameRow, bar, freeLabel)))
	}
	sidebar := container.NewVScroll(sidebarItems)
	sidebar.SetMinSize(fyne.NewSize(180, 0))
	previewTitle := widget.NewLabelWithStyle("Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	previewBox := canvas.NewRectangle(color.NRGBA{R: 0xe8, G: 0xea, B: 0xff, A: 0xff})
	previewBox.SetMinSize(fyne.NewSize(200, 200))
	previewIcon := widget.NewIcon(theme.FileImageIcon())
	previewImage := canvas.NewImageFromFile("")
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.SetMinSize(fyne.NewSize(200, 200))
	previewImage.Hide()
	previewName := widget.NewLabel("Select a file to preview")
	previewName.Alignment = fyne.TextAlignCenter
	previewName.Wrapping = fyne.TextWrapWord
	playPauseBtn := widget.NewButtonWithIcon("", theme.NewPrimaryThemedResource(theme.MediaPlayIcon()), nil)
	playPauseBtn.Importance = widget.LowImportance
	stopBtn := widget.NewButtonWithIcon("", theme.NewPrimaryThemedResource(theme.MediaStopIcon()), nil)
	stopBtn.Importance = widget.LowImportance
	stopBtn.Disable()
	seekSlider := widget.NewSlider(0, 1)
	timeLabel := widget.NewLabel("0:00 / 0:00")
	timeLabel.Alignment = fyne.TextAlignCenter
	audioControls := container.NewVBox(
		container.NewCenter(container.NewHBox(playPauseBtn, stopBtn)),
		seekSlider,
		timeLabel,
	)
	audioControls.Hide()
	loaded := false
	preview := container.NewVBox(
		previewTitle,
		container.NewCenter(container.NewStack(previewBox, container.NewCenter(previewIcon), previewImage)),
		previewName,
		audioControls,
	)
	previewWrap := container.NewVScroll(preview)
	previewWrap.SetMinSize(fyne.NewSize(220, 0))
	refreshControls := func() {
		playing, pos, dur := player.State()
		if playing {
		playPauseBtn.SetIcon(theme.NewPrimaryThemedResource(theme.MediaPauseIcon()))
		} else {
		playPauseBtn.SetIcon(theme.NewPrimaryThemedResource(theme.MediaPlayIcon()))
		}
		if dur > 0 {
			seekSlider.Max = dur
		}
		seekSlider.Value = pos
		seekSlider.Refresh()
		timeLabel.SetText(fmt.Sprintf("%s / %s", formatDuration(pos), formatDuration(dur)))
	}
	resetTransport := func() {
		playPauseBtn.SetIcon(theme.NewPrimaryThemedResource(theme.MediaPlayIcon()))
		stopBtn.Disable()
		seekSlider.Value = 0
		seekSlider.Max = 1
		seekSlider.Refresh()
		timeLabel.SetText("0:00 / 0:00")
		loaded = false
	}
	player = newMpvPlayer(
		func() { fyne.Do(refreshControls) },
		func() {
			fyne.Do(func() {
				player.Stop()
				resetTransport()
			})
		},
	)
	playPauseBtn.OnTapped = func() {
		if selectedPath == "" {
			return
		}
		if !loaded {
			if err := player.Play(selectedPath); err != nil {
				status.SetText(err.Error())
				return
			}
			loaded = true
			stopBtn.Enable()
			return
		}
		player.TogglePause()
	}
	stopBtn.OnTapped = func() {
		player.Stop()
		resetTransport()
	}
	seekSlider.OnChangeEnded = func(v float64) {
		if loaded {
			player.SeekTo(v)
		}
	}
	selectFile = func(f fileEntry) {
		if selectedPath != f.path {
			player.Stop()
		}
		selectedPath = f.path
		previewName.SetText(f.name)
		previewImage.Image = nil
		previewImage.Hide()
		previewIcon.Show()
		previewIcon.SetResource(iconFor(f.kind))
		switch f.kind {
		case "image":
			previewImage.Image = nil
			previewImage.File = f.path
			previewImage.Refresh()
			previewIcon.Hide()
			previewImage.Show()
		case "audio":
			path := f.path
			go func() {
				art := loadAlbumArt(path)
				if art == nil {
					return
				}
				fyne.Do(func() {
					if selectedPath != path {
						return
					}
					previewImage.File = ""
					previewImage.Image = art
					previewImage.Refresh()
					previewIcon.Hide()
					previewImage.Show()
				})
			}()
		}
		if f.kind == "audio" {
			resetTransport()
			audioControls.Show()
		} else {
			audioControls.Hide()
		}
	}
	showFileContextMenu = func(f fileEntry, e *fyne.PointEvent) {
		trash := fyne.NewMenuItem("Move to Trash", func() {
			if selectedPath == f.path {
				player.Stop()
				selectedPath = ""
			}
			if err := moveToTrash(f.path); err != nil {
				status.SetText(fmt.Sprintf("Cannot trash %s: %v", f.name, err))
				return
			}
			doRefresh()
		})
		widget.NewPopUpMenu(fyne.NewMenu("", trash), w.Canvas()).ShowAtPosition(e.AbsolutePosition)
	}
	statusBar := card(status)
	mainSplit := container.NewBorder(toolbar, statusBar, sidebar, previewWrap, fileArea)
	w.SetContent(container.NewPadded(mainSplit))
	w.SetOnClosed(func() { player.Stop() })
	startPath := home
	if d := filepath.Join(home, "Desktop"); dirExists(d) {
		startPath = d
	}
	navigateTo(startPath)
	w.ShowAndRun()
}
