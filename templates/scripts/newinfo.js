function editEntry(entryId) {
    const entry = document.getElementById(`entry-${entryId}`);

    const name = entry.querySelector("b").textContent.split(" (")[0].trim();
    const comment = entry.querySelector("i").textContent.replace(/"/g, "");
    const watched = entry.querySelector("b").textContent.includes("✓");

    entry.innerHTML = `
        <form onsubmit="saveEntry(event, ${entryId})">
            <input type="text" name="name" id="name" value="${name}" />
            <textarea name="comment" id="comment">${comment}</textarea>
            <label>
                Watched:
                <input type="checkbox" id="watched" name="watched" ${watched ? "checked" : ""} />
            </label>
            <button type="submit">Save</button>
            <button type="button" onclick="cancelEdit('${entryId}', '${name}', '${comment}', '${watched}')">Cancel</button>
        </form>
    `;
}

function saveEntry(event, entryId) {
    event.preventDefault();

    const currentUrl = window.location.href;
    const id = currentUrl.split('/').pop();

    const form = event.target;
    const name = form.name.value;
    const comment = form.comment.value;
    const watched = form.watched.checked;

    const payload = {
        name: name,
        comment: comment,
        watched: watched
    };

    fetch(`/films/${id}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(payload)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to update the entry');
        }
        return response.json();
    })
    .then(updatedData => {
        const entry = document.getElementById(`entry-${entryId}`);
        entry.innerHTML = `
            <b>${updatedData.name} ${updatedData.watched ? "(✓)" : "(✗)"}:</b> <i>"${updatedData.comment}"</i>
            <button class="edit-button" onclick="editEntry(${entryId})">Edit</button>
            <button class="delete-button" onclick="deleteEntry(1)">Delete</button>
        `;
    })
    .catch(error => {
        console.error('Error updating the entry:', error);
        alert('Failed to save changes. Please try again.');
    });
}

function cancelEdit(entryId, originalName, originalComment, originalWatched) {
    const entry = document.getElementById(`entry-${entryId}`);
    entry.innerHTML = `
        <b>${originalName} ${originalWatched ? "(✓)" : "(✗)"}:</b> <i>"${originalComment}"</i>
        <button class="edit-button" onclick="editEntry(${entryId})">Edit</button>
        <button class="delete-button" onclick="deleteEntry(1)">Delete</button>
    `;
}


function deleteEntry(entryId) {
    const currentUrl = window.location.href;
    const movieId = currentUrl.split('/').pop();
    if (!confirm("Are you sure you want to delete this entry?")) {
        return;
    }

    fetch(`/films/${movieId}`, {
        method: 'DELETE'
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to delete the entry');
        }
        return response.text();
    })
    .then(() => {
        const entry = document.getElementById(`entry-${entryId}`);
        entry.remove();
        alert('Entry deleted successfully!');
    })
    .catch(error => {
        console.error('Error deleting the entry:', error);
        alert('Failed to delete the entry. Please try again.');
    });
}

document.addEventListener('DOMContentLoaded', function () {
    document.getElementById('menu-search-bar').addEventListener('submit', function (event) {
        event.preventDefault();

        const searchInput = document.getElementById('search-input').value.trim();

        const imdbIdPattern = /^tt\d{7,8}$/;
        console.log('Test JS Loaded');

        if (imdbIdPattern.test(searchInput)) {
            window.location.href = `/films/${searchInput}`;
        } else {
            window.alert("invalid imdb id");
        }
    });
});
