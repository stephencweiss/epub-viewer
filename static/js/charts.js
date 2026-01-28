// Chart.js helpers for EPUB Reader Dashboard

async function loadRadarChart(elementId, url) {
    const response = await fetch(url);
    const data = await response.json();

    new Chart(document.getElementById(elementId), {
        type: 'radar',
        data: {
            labels: data.labels,
            datasets: [
                {
                    label: data.author1.label,
                    data: data.author1.data,
                    borderColor: 'rgb(54, 162, 235)',
                    backgroundColor: 'rgba(54, 162, 235, 0.2)',
                },
                {
                    label: data.author2.label,
                    data: data.author2.data,
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.2)',
                }
            ]
        },
        options: {
            scales: {
                r: { beginAtZero: true, max: 100 }
            }
        }
    });
}
