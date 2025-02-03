function showCitation(event, element) {
    const table = element.querySelector('.citation-table');
    if (table) {
        table.style.display = 'block';
        updateCitationPosition(event, table);
    }
}

function hideCitation(element) {
    const table = element.querySelector('.citation-table');
    if (table) {
        table.style.display = 'none';
    }
}

function updateCitationPosition(event, table) {
    const padding = 10;
    table.style.left = (event.pageX + padding) + 'px';
    table.style.top = (event.pageY + padding) + 'px';
}

document.addEventListener('mousemove', function(event) {
    const activeTable = document.querySelector('.citation-table[style*="display: block"]');
    if (activeTable) {
        updateCitationPosition(event, activeTable);
    }
});
