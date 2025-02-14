package store

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jhachmer/gomovie/internal/types"
	"github.com/jhachmer/gomovie/internal/util"
)

type MovieStore interface {
	CreateMovie(movie *types.Movie) (*types.Movie, error)
	UpdateMovie(movie *types.Movie) (*types.Movie, error)
	DeleteMovie(movieID string) error
	GetMovieByID(movieID string) (*types.Movie, error)
	GetAllMovies() ([]*types.MovieInfoData, error)
	SearchMovie(params types.SearchParams) ([]*types.MovieInfoData, error)
}

func (s *SQLiteStorage) CreateMovie(m *types.Movie) (*types.Movie, error) {
	_, err := s.DB.Exec( /*sql*/ `
		INSERT INTO movies (id, title, year, director, runtime, rated, released, plot, poster)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, m.ImdbID, m.Title, m.Year, m.Director, m.Runtime, m.Rated, m.Released, m.Plot, m.Poster)
	if err != nil {
		return nil, err
	}
	err = s.createMovieRatings(m)
	if err != nil {
		return nil, err
	}
	err = s.createMovieGenres(m)
	if err != nil {
		return nil, err
	}
	err = s.createMovieActors(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SQLiteStorage) CreateMovieTx(tx *sql.Tx, m *types.Movie) (*types.Movie, error) {
	_, err := tx.Exec( /*sql*/ `
		INSERT INTO movies (id, title, year, director, runtime, rated, released, plot, poster)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, m.ImdbID, m.Title, m.Year, m.Director, m.Runtime, m.Rated, m.Released, m.Plot, m.Poster)
	if err != nil {
		return nil, err
	}
	err = s.createMovieRatingsTx(tx, m)
	if err != nil {
		return nil, err
	}
	err = s.createMovieGenresTx(tx, m)
	if err != nil {
		return nil, err
	}
	err = s.createActorsTx(tx, m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SQLiteStorage) UpdateMovie(m *types.Movie) (*types.Movie, error) {
	_, err := s.DB.Exec( /*sql*/ `
	UPDATE movies
	SET title = ?, year = ?, director = ?, runtime = ?, rated = ?, released = ?, plot = ?, poster = ?
	WHERE id = ?;
	`, m.Title, m.Year, m.Director, m.Runtime, m.Rated, m.Released, m.Plot, m.Poster, m.ImdbID)
	if err != nil {
		return nil, err
	}
	err = s.updateRatings(m)
	if err != nil {
		return nil, err
	}
	err = s.updateGenres(m)
	if err != nil {
		return nil, err
	}
	err = s.updateActors(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SQLiteStorage) DeleteMovie(imdbId string) error {
	_, err := s.DB.Exec( /*sql*/ `
	DELETE FROM movies WHERE id = ?;
	`, imdbId)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStorage) updateRatings(m *types.Movie) error {
	for _, rating := range m.Ratings {
		_, err := s.DB.Exec( /*sql*/ `
			UPDATE ratings
			SET value = ?
			WHERE movie_id = ? AND source = ?;
			`, rating.Value, m.ImdbID, rating.Source)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) updateGenres(m *types.Movie) error {
	genres := util.SplitIMDBString(m.Genre)

	rows, err := s.DB.Query( /*sql*/ `
	SELECT g.name
	FROM genres g
	INNER JOIN movies_genres mg ON g.id = mg.genre_id
	WHERE mg.movie_id = ?`, m.ImdbID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var genresFromDB []string
	for rows.Next() {
		var genre string
		if err := rows.Scan(&genre); err != nil {
			return err
		}
		genresFromDB = append(genresFromDB, genre)
	}

	for _, g := range genresFromDB {
		if !slices.Contains(genres, g) {
			_, err := s.DB.Exec( /*sql*/ `
			DELETE FROM movies_genres
			WHERE
			movie_id = ?
			AND
			genre_id = (SELECT id FROM genres WHERE name = ?);
			`, m.ImdbID, g)
			if err != nil {
				return err
			}
		}
	}

	for _, genre := range genres {
		var genreID int64
		err := s.DB.QueryRow( /*sql*/ `
			SELECT id
			FROM genres
			WHERE name = ?;
			`, genre).Scan(&genreID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := s.DB.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO genres (name)
       			VALUES (?);
       			`, genre)
			if err != nil {
				return err
			}
			genreID, _ = res.LastInsertId()
		}
		_, err = s.DB.Exec( /*sql*/ `
			INSERT OR IGNORE
       		INTO movies_genres (movie_id, genre_id)
       		VALUES (?, ?);
       		`, m.ImdbID, genreID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) updateActors(m *types.Movie) error {
	actors := util.SplitIMDBString(m.Actors)

	rows, err := s.DB.Query( /*sql*/ `
	SELECT a.name
	FROM actors a
	INNER JOIN movies_actors ma ON a.id = ma.actor_id
	WHERE ma.movie_id = ?`, m.ImdbID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var actorsFromDB []string
	for rows.Next() {
		var actor string
		if err := rows.Scan(&actor); err != nil {
			return err
		}
		actorsFromDB = append(actorsFromDB, actor)
	}

	for _, a := range actorsFromDB {
		if !slices.Contains(actors, a) {
			_, err := s.DB.Exec( /*sql*/ `
			DELETE FROM movies_actors
			WHERE
			movie_id = ?
			AND
			actor_id = (SELECT id FROM actors WHERE name = ?);
			`, m.ImdbID, a)
			if err != nil {
				return err
			}
		}
	}

	for _, actor := range actors {
		var actorID int64
		err := s.DB.QueryRow( /*sql*/ `
			SELECT id
			FROM actors
			WHERE name = ?;
			`, actor).Scan(&actorID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := s.DB.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO actors (name)
       			VALUES (?);
       			`, actor)
			if err != nil {
				return err
			}
			actorID, _ = res.LastInsertId()
		}
		_, err = s.DB.Exec( /*sql*/ `
			INSERT OR IGNORE
       		INTO movies_actors (movie_id, actor_id)
       		VALUES (?, ?);
       		`, m.ImdbID, actorID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) GetMovieByID(movieID string) (*types.Movie, error) {
	var movie types.Movie

	err := s.DB.QueryRow( /*sql*/ `
        SELECT
            id, title, year, rated, released, runtime, plot, poster, director
        FROM movies
        WHERE id = ?`, movieID).Scan(
		&movie.ImdbID, &movie.Title, &movie.Year, &movie.Rated,
		&movie.Released, &movie.Runtime, &movie.Plot, &movie.Poster, &movie.Director)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query( /*sql*/ `
        SELECT g.name
        FROM genres g
        INNER JOIN movies_genres mg ON g.id = mg.genre_id
        WHERE mg.movie_id = ?`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var genre string
		if err := rows.Scan(&genre); err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}
	movie.Genre = strings.Join(genres, ", ")

	rows, err = s.DB.Query( /*sql*/ `
        SELECT a.name
        FROM actors a
        INNER JOIN movies_actors ma ON a.id = ma.actor_id
        WHERE ma.movie_id = ?`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actors []string
	for rows.Next() {
		var actor string
		if err := rows.Scan(&actor); err != nil {
			return nil, err
		}
		actors = append(actors, actor)
	}
	movie.Actors = strings.Join(actors, ", ")

	rows, err = s.DB.Query( /*sql*/ `
        SELECT source, value
        FROM ratings
        WHERE movie_id = ?`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []types.Rating
	for rows.Next() {
		var rating types.Rating
		if err := rows.Scan(&rating.Source, &rating.Value); err != nil {
			return nil, err
		}
		ratings = append(ratings, rating)
	}
	movie.Ratings = ratings

	return &movie, nil
}

func (s *SQLiteStorage) GetAllMovies() ([]*types.MovieInfoData, error) {
	var movies []*types.MovieInfoData
	rows, err := s.DB.Query( /*sql*/ `
        SELECT id
        FROM movies
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movieIDs []string
	for rows.Next() {
		var movieID string
		if err := rows.Scan(&movieID); err != nil {
			return nil, err
		}
		movieIDs = append(movieIDs, movieID)
	}
	for _, movieID := range movieIDs {
		movie, err := s.GetMovieByID(movieID)
		if err != nil {
			return nil, err
		}
		entries, err := s.GetEntries(movieID)
		if err != nil {
			return nil, err
		}
		data := types.MovieInfoData{Movie: movie, Entry: entries}
		movies = append(movies, &data)
	}
	return movies, nil
}

func (s *SQLiteStorage) SearchMovie(params types.SearchParams) ([]*types.MovieInfoData, error) {
	filters := []string{}
	args := []any{}

	if len(params.Genres) > 0 {
		genrePlaceholders := make([]string, len(params.Genres))
		for i, genre := range params.Genres {
			genrePlaceholders[i] = "?"
			args = append(args, "%"+genre+"%")
		}
		filters = append(filters, "g.name LIKE "+strings.Join(genrePlaceholders, " OR g.name LIKE "))
	}

	if len(params.Actors) > 0 {
		actorPlaceholders := make([]string, len(params.Actors))
		for i, actor := range params.Actors {
			actorPlaceholders[i] = "?"
			args = append(args, "%"+actor+"%")
		}
		filters = append(filters, "a.name LIKE "+strings.Join(actorPlaceholders, " OR a.name LIKE "))
	}

	if params.Years.StartYear != "" && params.Years.EndYear != "" {
		filters = append(filters, "m.year BETWEEN ? AND ?")
		args = append(args, params.Years.StartYear, params.Years.EndYear)
	}

	query := /*sql*/ `
		SELECT DISTINCT m.id
		FROM movies m
		LEFT JOIN movies_genres mg ON m.id = mg.movie_id
		LEFT JOIN genres g ON mg.genre_id = g.id
		LEFT JOIN movies_actors ma ON m.id = ma.movie_id
		LEFT JOIN actors a ON ma.actor_id = a.id
		`

	if len(filters) > 0 {
		query += "WHERE " + strings.Join(filters, " AND ")
	}

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	results := []*types.MovieInfoData{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		mov, err := s.GetMovieByID(id)
		if err != nil {
			return nil, err
		}
		entry, err := s.GetEntries(id)
		if err != nil {
			return nil, err
		}
		results = append(results, &types.MovieInfoData{Movie: mov, Entry: entry})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

func (s *SQLiteStorage) createMovieRatings(m *types.Movie) error {
	for _, rating := range m.Ratings {
		_, err := s.DB.Exec( /*sql*/ `
			INSERT INTO ratings (movie_id, source, value)
			VALUES (?, ?, ?);
			`, m.ImdbID, rating.Source, rating.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) createMovieRatingsTx(tx *sql.Tx, m *types.Movie) error {
	for _, rating := range m.Ratings {
		_, err := tx.Exec( /*sql*/ `
			INSERT INTO ratings (movie_id, source, value)
			VALUES (?, ?, ?);
			`, m.ImdbID, rating.Source, rating.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) createMovieGenres(m *types.Movie) error {
	genres := util.SplitIMDBString(m.Genre)
	for _, genre := range genres {
		var genreID int64
		err := s.DB.QueryRow( /*sql*/ `
			SELECT id
			FROM genres
			WHERE name = ?;
			`, genre).Scan(&genreID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := s.DB.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO genres (name)
       			VALUES (?);
       			`, genre)
			if err != nil {
				return err
			}
			genreID, _ = res.LastInsertId()
		}
		_, err = s.DB.Exec( /*sql*/ `INSERT OR IGNORE
       		INTO movies_genres (movie_id, genre_id)
       		VALUES (?, ?);
       		`, m.ImdbID, genreID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) createMovieGenresTx(tx *sql.Tx, m *types.Movie) error {
	genres := util.SplitIMDBString(m.Genre)
	for _, genre := range genres {
		var genreID int64
		err := tx.QueryRow( /*sql*/ `
			SELECT id
			FROM genres
			WHERE name = ?;
			`, genre).Scan(&genreID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO genres (name)
       			VALUES (?);
       			`, genre)
			if err != nil {
				return err
			}
			genreID, _ = res.LastInsertId()
		}
		_, err = tx.Exec( /*sql*/ `INSERT OR IGNORE
       		INTO movies_genres (movie_id, genre_id)
       		VALUES (?, ?);
       		`, m.ImdbID, genreID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) createMovieActors(m *types.Movie) error {
	actors := util.SplitIMDBString(m.Actors)
	for _, actor := range actors {
		var actorID int64
		err := s.DB.QueryRow( /*sql*/ `
			SELECT id
			FROM actors
			WHERE name = ?;
			`, actor).Scan(&actorID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := s.DB.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO actors (name)
       			VALUES (?);
       			`, actor)
			if err != nil {
				return err
			}
			actorID, _ = res.LastInsertId()
		}
		_, err = s.DB.Exec( /*sql*/ `
			INSERT OR IGNORE
       		INTO movies_actors (movie_id, actor_id)
       		VALUES (?, ?);
			`, m.ImdbID, actorID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) createActorsTx(tx *sql.Tx, m *types.Movie) error {
	actors := util.SplitIMDBString(m.Actors)
	for _, actor := range actors {
		var actorID int64
		err := tx.QueryRow( /*sql*/ `
			SELECT id
			FROM actors
			WHERE name = ?;
			`, actor).Scan(&actorID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.Exec( /*sql*/ `
				INSERT OR IGNORE
       			INTO actors (name)
       			VALUES (?);
       			`, actor)
			if err != nil {
				return err
			}
			actorID, _ = res.LastInsertId()
		}
		_, err = tx.Exec( /*sql*/ `
			INSERT OR IGNORE
       		INTO movies_actors (movie_id, actor_id)
       		VALUES (?, ?);
			`, m.ImdbID, actorID)
		if err != nil {
			return err
		}
	}
	return nil
}
