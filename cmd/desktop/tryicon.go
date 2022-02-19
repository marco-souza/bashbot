package desktop

import (
	"fmt"
	"time"

	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"github.com/marco-souza/binance-dashboard/cmd"
)

func StartApp() {
	onExit := func() {
		fmt.Println("Exiting...")
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	// Sys tray icon
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("Awesome App")
	systray.SetTooltip("Lantern")

	// Menu items
	mAccountSnapshot := systray.AddMenuItem("Account snapshot", "")
	mFetchAsset := systray.AddMenuItem("Fetch BTC-USD", "recurrent")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item. Only available on Mac.
	mQuit.SetIcon(icon.Data)

	registerClickHandlers := func() {
		for {
			select {
			// TODO: show window
			case <-mAccountSnapshot.ClickedCh:
				handleAccountSnapshot()
			case <-mFetchAsset.ClickedCh:
				go handleFetchTicker(mFetchAsset)
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}
	go registerClickHandlers()
}
func handleFetchTicker(item *systray.MenuItem) {
	for {
		time.Sleep(2 * time.Second)
		tiker := cmd.FetchTicker("BTCUSDT")
		item.SetTitle(tiker.Price)
	}
}

func handleAccountSnapshot() {
	res := cmd.FetchAccountSnapshot()
	for _, dp := range res.SnapshotVos {
		t := time.Unix(int64(dp.UpdateTime/1000), 0)
		fmt.Println(" -> ", t, dp.Data.TotalBtcAsset)

		for _, currency := range dp.Data.Balances {
			fmt.Printf("\t%s $%s\n", currency.Asset, currency.Free)
		}
	}
}