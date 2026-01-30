// Table sorting functionality for EPUB Reader Dashboard
// Click column headers to sort. Click again to reverse.

function initSortableTables() {
    document.querySelectorAll('table[data-sortable]').forEach(table => {
        initSortableTable(table);
    });
}

function initSortableTable(table) {
    const headers = table.querySelectorAll('th[data-sort]');
    headers.forEach((header, index) => {
        header.style.cursor = 'pointer';
        header.setAttribute('title', 'Click to sort');

        // Add sort indicator span
        if (!header.querySelector('.sort-indicator')) {
            const indicator = document.createElement('span');
            indicator.className = 'sort-indicator';
            indicator.style.marginLeft = '4px';
            indicator.style.opacity = '0.5';
            header.appendChild(indicator);
        }

        header.addEventListener('click', () => {
            sortTable(table, header, index);
        });
    });
}

function sortTable(table, header, columnIndex) {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    const type = header.dataset.type || 'string';

    // Determine sort direction
    const currentDir = header.dataset.sortDir || 'none';
    const newDir = currentDir === 'asc' ? 'desc' : 'asc';

    // Reset all header indicators
    table.querySelectorAll('th[data-sort]').forEach(th => {
        th.dataset.sortDir = 'none';
        const ind = th.querySelector('.sort-indicator');
        if (ind) ind.textContent = '';
    });

    // Set current header
    header.dataset.sortDir = newDir;
    const indicator = header.querySelector('.sort-indicator');
    if (indicator) {
        indicator.textContent = newDir === 'asc' ? ' \u25B2' : ' \u25BC';
    }

    // Sort rows
    rows.sort((a, b) => {
        const cellA = a.cells[columnIndex];
        const cellB = b.cells[columnIndex];

        // Use data-value attribute if present, otherwise use text content
        let valA = cellA.dataset.value !== undefined ? cellA.dataset.value : cellA.textContent.trim();
        let valB = cellB.dataset.value !== undefined ? cellB.dataset.value : cellB.textContent.trim();

        // Handle empty/dash values
        if (valA === '-' || valA === '') valA = type === 'number' ? '-999999' : 'zzz';
        if (valB === '-' || valB === '') valB = type === 'number' ? '-999999' : 'zzz';

        let comparison = 0;
        if (type === 'number') {
            comparison = parseFloat(valA) - parseFloat(valB);
        } else {
            comparison = valA.localeCompare(valB, undefined, { sensitivity: 'base' });
        }

        return newDir === 'asc' ? comparison : -comparison;
    });

    // Re-append sorted rows
    rows.forEach(row => tbody.appendChild(row));
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', initSortableTables);

// Re-initialize after HTMX swaps (for dynamic content)
document.body.addEventListener('htmx:afterSwap', initSortableTables);
