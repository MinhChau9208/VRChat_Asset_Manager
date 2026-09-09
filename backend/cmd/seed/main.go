package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"vrchat-asset-manager/backend/internal/asset"
	"vrchat-asset-manager/backend/internal/database"
	"vrchat-asset-manager/backend/migrations"
)

func main() {
	dbPath := filepath.Join("..", "data", "app.db")
	if _, err := os.Stat(dbPath); err != nil {
		dbPath = filepath.Join("data", "app.db")
	}

	absPath, _ := filepath.Abs(dbPath)
	fmt.Printf("Seeding sample assets into SQLite: %s\n", absPath)

	db, err := database.Connect(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer db.Close()

	if err := db.Migrate(migrations.FS); err != nil {
		log.Fatalf("Failed to migrate database: %v\n", err)
	}

	repo := asset.NewRepository(db.DB)
	ctx := context.Background()

	// Check existing assets count
	existing, err := repo.List(ctx, asset.FilterParams{})
	if err != nil {
		log.Fatalf("Failed to query assets: %v\n", err)
	}

	if len(existing) > 0 {
		fmt.Printf("Database already contains %d asset(s). Skipping seed to avoid duplicates.\n", len(existing))
		return
	}

	catAvatar := int64(1)
	catHair := int64(2)
	catClothes := int64(3)
	catShoes := int64(4)
	catGimmick := int64(6)

	samples := []asset.CreateAssetRequest{
		{
			Name:        "Manuka Avatar",
			CategoryID:  &catAvatar,
			Author:      "Jingo Channel",
			BoothURL:    "https://booth.pm/en/items/4394473",
			LocalPath:   "D:/VRChat/Avatars/Manuka",
			Description: "Popular base model avatar for VRChat with high customizability.",
			Tags:        []string{"avatar", "female", "physbone"},
		},
		{
			Name:        "Fluffy Twin Tails",
			CategoryID:  &catHair,
			Author:      "Sisters Hair",
			BoothURL:    "https://booth.pm/en/items/4567890",
			LocalPath:   "D:/VRChat/Hair/TwinTails",
			Description: "Soft textured twin tail hairstyle with dynamic PhysBones.",
			Tags:        []string{"hair", "twintails", "cute"},
		},
		{
			Name:        "Cyberpunk Streetwear Jacket",
			CategoryID:  &catClothes,
			Author:      "Virtual Threads",
			BoothURL:    "https://booth.pm/en/items/3214567",
			LocalPath:   "D:/VRChat/Clothes/TechwearJacket",
			Description: "Oversized techwear style jacket with glowing emissive materials.",
			Tags:        []string{"clothes", "streetwear", "unisex"},
		},
		{
			Name:        "High-top Combat Boots",
			CategoryID:  &catShoes,
			Author:      "KicksVR",
			BoothURL:    "https://booth.pm/en/items/7891234",
			LocalPath:   "D:/VRChat/Shoes/CombatBoots",
			Description: "Sturdy combat boots rigged for multiple avatar bases.",
			Tags:        []string{"shoes", "boots", "punk"},
		},
		{
			Name:        "Animated Fox Ears & Tail",
			CategoryID:  &catGimmick,
			Author:      "Kitsune Workshop",
			BoothURL:    "https://booth.pm/en/items/9876543",
			LocalPath:   "D:/VRChat/Gimmicks/FoxEarsTail",
			Description: "Expressive animated animal ears and tail accessory package.",
			Tags:        []string{"gimmick", "ears", "tail", "physbone"},
		},
	}

	for _, s := range samples {
		created, err := repo.Create(ctx, s)
		if err != nil {
			log.Fatalf("Failed to create sample asset %q: %v\n", s.Name, err)
		}
		fmt.Printf("Created sample asset [%d]: %s (Category: %d, Tags: %v)\n", created.ID, created.Name, *created.CategoryID, created.Tags)
	}

	fmt.Println("Seeding completed successfully! 5 sample assets created.")
}
