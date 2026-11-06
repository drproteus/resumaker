// Command admin manages per-user feature flags. It runs on the server with
// RESUMAKER_DATABASE_URL set, for example:
//
//	admin users
//	admin features list --email jane@example.com
//	admin features enable gemini --email jane@example.com
//	admin features disable gemini --user-id 0b6f...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/store"
)

const usage = `usage:
  admin users
  admin features list    (--email ADDRESS | --user-id ID)
  admin features enable  FEATURE (--email ADDRESS | --user-id ID)
  admin features disable FEATURE (--email ADDRESS | --user-id ID)

features: ` + "%s"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "admin:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return usageError()
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(out, usageText())
		return nil
	}

	dbURL := os.Getenv("RESUMAKER_DATABASE_URL")
	if dbURL == "" {
		return errors.New("RESUMAKER_DATABASE_URL is required")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	switch args[0] {
	case "users":
		return listUsers(ctx, st, out)
	case "features":
		return manageFeatures(ctx, st, args[1:], out)
	default:
		return usageError()
	}
}

func usageText() string {
	return fmt.Sprintf(usage, strings.Join(app.FeatureNames(), ", "))
}

func usageError() error {
	return errors.New(usageText())
}

func listUsers(ctx context.Context, st *store.Store, out io.Writer) error {
	accounts, err := st.ListUserAccounts(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "ID\tNAME\tEMAILS\tFEATURES")
	for _, a := range accounts {
		features, err := st.ListFeatures(ctx, a.ID)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", a.ID, a.DisplayName,
			orNone(strings.Join(a.Emails, ", ")), orNone(strings.Join(features, ", ")))
	}
	return nil
}

func manageFeatures(ctx context.Context, st *store.Store, args []string, out io.Writer) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "list":
		user, err := resolveUser(ctx, st, args[1:])
		if err != nil {
			return err
		}
		features, err := st.ListFeatures(ctx, user.ID)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, orNone(strings.Join(features, ", ")))
		return nil
	case "enable", "disable":
		if len(args) < 2 {
			return fmt.Errorf("%s needs a feature name: %s", args[0], strings.Join(app.FeatureNames(), ", "))
		}
		feature := args[1]
		if !app.ValidFeature(feature) {
			return fmt.Errorf("unknown feature %q: choose from %s", feature, strings.Join(app.FeatureNames(), ", "))
		}
		user, err := resolveUser(ctx, st, args[2:])
		if err != nil {
			return err
		}
		enabled := args[0] == "enable"
		if err := st.SetFeature(ctx, user.ID, feature, enabled); err != nil {
			return err
		}
		state := "disabled"
		if enabled {
			state = "enabled"
		}
		fmt.Fprintf(out, "%s %s for %s (%s)\n", feature, state, user.DisplayName, user.ID)
		return nil
	default:
		return usageError()
	}
}

// resolveUser finds the account named by --email or --user-id.
func resolveUser(ctx context.Context, st *store.Store, args []string) (store.User, error) {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	email := fs.String("email", "", "email address linked to the account")
	id := fs.String("user-id", "", "account ID, as shown by admin users")
	if err := fs.Parse(args); err != nil {
		return store.User{}, err
	}
	if fs.NArg() > 0 {
		return store.User{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}

	switch {
	case *email != "" && *id != "":
		return store.User{}, errors.New("use either --email or --user-id, not both")
	case *email != "":
		user, err := st.UserByEmail(ctx, *email)
		if errors.Is(err, store.ErrNotFound) {
			return store.User{}, fmt.Errorf("no account with email %s", *email)
		}
		return user, err
	case *id != "":
		userID, err := uuid.Parse(*id)
		if err != nil {
			return store.User{}, fmt.Errorf("invalid --user-id: %w", err)
		}
		user, err := st.UserByID(ctx, userID)
		if errors.Is(err, store.ErrNotFound) {
			return store.User{}, fmt.Errorf("no account with ID %s", userID)
		}
		return user, err
	default:
		return store.User{}, errors.New("choose the account with --email or --user-id")
	}
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
