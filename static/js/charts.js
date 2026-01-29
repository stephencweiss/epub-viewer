// Chart.js helpers for EPUB Reader Dashboard

async function loadRadarChart(elementId, url) {
    try {
        const canvas = document.getElementById(elementId);
        if (!canvas) {
            console.error('Canvas element not found:', elementId);
            return;
        }

        if (typeof Chart === 'undefined') {
            console.error('Chart.js not loaded');
            return;
        }

        const response = await fetch(url);
        if (!response.ok) {
            console.error('API error:', response.status, response.statusText);
            return;
        }

        const data = await response.json();
        if (data.error) {
            console.error('API returned error:', data.error);
            return;
        }

        new Chart(canvas, {
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
    } catch (err) {
        console.error('Failed to load radar chart:', err);
    }
}
