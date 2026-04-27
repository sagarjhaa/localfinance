package insights

import (
	"fmt"
	"time"

	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

// fixedNow is the pinned "current time" used by all tests. Picked as a Wednesday
// so day-of-week tests don't accidentally land on a boundary.
var fixedNow = time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC) // Wednesday

func dayOffset(days int) time.Time {
	return fixedNow.AddDate(0, 0, -days)
}

func tx(id string, daysAgo int, desc, category string, amount float64) models.TransactionRef {
	return models.TransactionRef{
		ID:          id,
		Date:        dayOffset(daysAgo).Format(time.RFC3339),
		Description: desc,
		Category:    category,
		Amount:      amount,
		Type:        "expense",
	}
}

// makeFixtureSet returns ~50 transactions designed to trigger every rule at least
// once. Used by the keys-stability test and a few rule tests.
func makeFixtureSet() []models.TransactionRef {
	out := []models.TransactionRef{}

	// CategoryShift: Dining surges in last 30d vs prior 60d
	// Prior 60d (days 31-90): 6 dining at ~$30 = $180 total ($90/30d-equiv)
	for i := 0; i < 6; i++ {
		out = append(out, tx(fmt.Sprintf("dn-prior-%d", i), 35+i*8, "Local Diner", "Dining", 30.0))
	}
	// Last 30d: 5 dining at $80 = $400
	for i := 0; i < 5; i++ {
		out = append(out, tx(fmt.Sprintf("dn-last-%d", i), 5+i*5, "Fancy Bistro", "Dining", 80.0))
	}

	// NewRecurringMerchant: "ChatGPT Plus" charges only in last 30d
	out = append(out, tx("gpt-1", 28, "ChatGPT Plus", "Software", 20.0))
	out = append(out, tx("gpt-2", 1, "ChatGPT Plus", "Software", 20.0))

	// DayOfWeekCluster: Coffee category, 12 transactions, mostly Saturdays
	// Pick saturdays. fixedNow=Wed Apr 22 2026; Saturday is daysAgo where (Wed - 4) → Saturday is 4 days ago.
	saturdays := []int{4, 11, 18, 25, 32, 39, 46, 53, 60, 67}
	for i, d := range saturdays {
		out = append(out, tx(fmt.Sprintf("cf-sat-%d", i), d, "Saturday Coffee Co", "Coffee", 25.0))
	}
	// 2 non-saturday small ones
	out = append(out, tx("cf-mon-1", 9, "Monday Mug", "Coffee", 5.0))
	out = append(out, tx("cf-tue-1", 8, "Tuesday Cup", "Coffee", 5.0))

	// PriceEscalation: "Gym Membership" rising prices
	out = append(out, tx("gym-1", 80, "Acme Gym", "Fitness", 30.0))
	out = append(out, tx("gym-2", 50, "Acme Gym", "Fitness", 35.0))
	out = append(out, tx("gym-3", 20, "Acme Gym", "Fitness", 50.0))

	// AnomalyCluster: Travel category baseline + 3 anomalies in same week
	for i := 0; i < 10; i++ {
		out = append(out, tx(fmt.Sprintf("tr-norm-%d", i), 40+i*5, "Local Bus", "Travel", 5.0))
	}
	out = append(out, tx("tr-anom-1", 15, "First Class Flight", "Travel", 2000.0))
	out = append(out, tx("tr-anom-2", 14, "Five Star Hotel", "Travel", 1800.0))
	out = append(out, tx("tr-anom-3", 12, "Yacht Rental", "Travel", 2500.0))

	return out
}
