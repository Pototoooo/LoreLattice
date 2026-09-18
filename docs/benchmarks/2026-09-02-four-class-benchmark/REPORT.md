# Four-class RAG / Wiki / Hybrid benchmark

Runs: 96; cases: 16; repeats: 2

| Category | Mode | Accuracy | Completeness | Retrieval recall | Citation recall | Rerank path | Mean / median tokens | P50 / P95 latency |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| overall | rag | 68.75% | 93.38% | 99.22% | 64.3% | 34.38% | 68733 / 44823 | 27.41s / 67.12s |
| overall | wiki | 65.62% | 94.71% | 93.31% | 88.8% | 0.0% | 58089 / 42888 | 34.51s / 78.17s |
| overall | hybrid | 71.88% | 95.78% | 95.23% | 58.05% | 6.25% | 115950 / 82996 | 38.55s / 79.52s |
| single_hop_factual | rag | 100.0% | 100.0% | 100.0% | 37.5% | 0.0% | 23928 / 23573 | 9.01s / 10.92s |
| single_hop_factual | wiki | 100.0% | 100.0% | 100.0% | 87.5% | 0.0% | 27351 / 21330 | 10.01s / 20.08s |
| single_hop_factual | hybrid | 100.0% | 100.0% | 100.0% | 50.0% | 0.0% | 29887 / 28730 | 9.96s / 13.12s |
| local_synthesis | rag | 100.0% | 100.0% | 100.0% | 75.0% | 37.5% | 28045 / 23618 | 23.28s / 28.87s |
| local_synthesis | wiki | 100.0% | 100.0% | 100.0% | 100.0% | 0.0% | 29556 / 26186 | 26.95s / 35.44s |
| local_synthesis | hybrid | 100.0% | 100.0% | 100.0% | 75.0% | 12.5% | 60918 / 67452 | 30.38s / 38.92s |
| multi_hop | rag | 37.5% | 83.75% | 100.0% | 85.0% | 0.0% | 70100 / 71525 | 32.78s / 49.70s |
| multi_hop | wiki | 62.5% | 94.09% | 89.17% | 88.33% | 0.0% | 64835 / 55227 | 38.86s / 47.81s |
| multi_hop | hybrid | 50.0% | 93.18% | 100.0% | 70.0% | 12.5% | 184090 / 111442 | 45.57s / 85.34s |
| global_thematic | rag | 37.5% | 89.76% | 96.88% | 59.69% | 100.0% | 152858 / 162111 | 58.60s / 82.48s |
| global_thematic | wiki | 0.0% | 84.74% | 84.06% | 79.38% | 0.0% | 110612 / 90756 | 54.03s / 82.50s |
| global_thematic | hybrid | 37.5% | 89.94% | 80.94% | 37.19% | 0.0% | 188907 / 210037 | 54.61s / 80.17s |

## Pairwise completeness
- overall wiki_vs_rag: 5W/23T/4L, all-comparison win rate=15.62%, delta=1.33 pp, bootstrap 95% CI=[-2.42, 6.3] pp
- overall hybrid_vs_rag: 6W/22T/4L, all-comparison win rate=18.75%, delta=2.4 pp, bootstrap 95% CI=[-1.79, 7.25] pp
- single_hop_factual wiki_vs_rag: 0W/8T/0L, all-comparison win rate=0.0%, delta=0 pp, bootstrap 95% CI=[0, 0] pp
- single_hop_factual hybrid_vs_rag: 0W/8T/0L, all-comparison win rate=0.0%, delta=0 pp, bootstrap 95% CI=[0, 0] pp
- local_synthesis wiki_vs_rag: 0W/8T/0L, all-comparison win rate=0.0%, delta=0 pp, bootstrap 95% CI=[0, 0] pp
- local_synthesis hybrid_vs_rag: 0W/8T/0L, all-comparison win rate=0.0%, delta=0 pp, bootstrap 95% CI=[0, 0] pp
- multi_hop wiki_vs_rag: 3W/5T/0L, all-comparison win rate=37.5%, delta=10.34 pp, bootstrap 95% CI=[1.14, 26.36] pp
- multi_hop hybrid_vs_rag: 4W/2T/2L, all-comparison win rate=50.0%, delta=9.43 pp, bootstrap 95% CI=[-2.16, 24.32] pp
- global_thematic wiki_vs_rag: 2W/2T/4L, all-comparison win rate=25.0%, delta=-5.02 pp, bootstrap 95% CI=[-12.16, 2.13] pp
- global_thematic hybrid_vs_rag: 2W/4T/2L, all-comparison win rate=25.0%, delta=0.18 pp, bootstrap 95% CI=[-10.89, 10.89] pp
