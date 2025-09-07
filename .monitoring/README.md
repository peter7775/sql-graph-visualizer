# SQL Graph Visualizer - Code Monitoring System

Tento monitoring system automaticky kontroluje potenciální neautorizované kopírování vašeho kódu na GitHubu a jiných platformách.

## 🚀 Rychlé spuštění

```bash
# Spusťte setup script pro automatickou instalaci
./.monitoring/setup-.monitoring.sh

# Nebo manuálně spusťte .monitoring
./.monitoring/code-.monitoring.sh
```

## 📁 Struktura souborů

```
monitoring/
├── README.md                    # Tato dokumentace
├── github-search-queries.md     # Seznam vyhledávacích dotazů
├── code-monitoring.sh           # Hlavní monitoring script
├── setup-monitoring.sh          # Instalační script
├── monitoring-results.log       # Log výsledků
└── results/                     # JSON výsledky z API
    ├── high_*.json             # Výsledky vysoké priority
    ├── med_*.json              # Výsledky střední priority
    └── repo_search.json        # Vyhledávání repozitářů
```

## 🔍 Co monitoring sleduje

### Vysoká priorita (denně)
- Copyright a licenční informace
- Unikátní názvy funkcí a struktur
- Specifické import paths
- Patent-pending komentáře

### Střední priorita (týdně)
- Architekturní vzory (DDD komentáře)
- Charakteristické log zprávy
- Specifické konfigurace

### Nízká priorita (měsíčně)
- Obecné názvy proměnných
- Běžné vzory kódu

## 📊 Jak používat výsledky

### Pozitivní nálezy (⚠️)
Když monitoring najde potenciální kopie:

1. **Zkontrolujte výsledky manuálně**
   ```bash
   cat .monitoring/.monitoring-results.log | grep ALERT
   ```

2. **Prohlédněte si nalezené repository**
   - Otevřete URL z log souboru
   - Ověřte, zda se jedná o skutečné kopírování
   - Zkontrolujte licenci a autorství

3. **Možné akce**
   - Kontaktujte autora neautorizované kopie
   - Podejte DMCA takedown notice
   - Konzultujte s právníkem ohledně porušení licence

### False positive
- Může najít vaše vlastní repository na různých účtech
- Může najít legitimní odkazy na vaši práci
- Vždy manuálně ověřte před akcí

## ⚙️ Konfigurace

### Změna frequency monitoringu

Editujte crontab:
```bash
crontab -e
```

Aktuální nastavení:
- **Denně v 9:00** - základní monitoring
- **Pondělí v 8:00** - detailní monitoring

### Customizace vyhledávacích dotazů

Editujte `monitoring/code-monitoring.sh` a upravte:
```bash
HIGH_PRIORITY_QUERIES=(
    # Přidejte vaše vlastní unikátní řetězce
    "\"Your Unique Function Name\""
)
```

### Email notifikace

Pro aktivaci email notifikací:
```bash
# Nainstalujte mail program
sudo pacman -S mailutils  # nebo odpovídající pro váš systém

# Script automaticky pošle email při detekci
```

## 🔧 Troubleshooting

### Chyba: "jq: command not found"
```bash
# Manjaro/Arch
sudo pacman -S jq

# Ubuntu/Debian  
sudo apt install jq

# CentOS/RHEL
sudo yum install jq
```

### Chyba: "curl: command not found"
```bash
# Většinou je curl již nainstalován, pokud ne:
sudo pacman -S curl  # Manjaro/Arch
```

### GitHub API rate limiting
- GitHub API povoluje 60 requestů/hodinu pro neautentizované requesty
- Pro více requestů vytvořte GitHub Personal Access Token
- Přidejte do script: `-H "Authorization: token YOUR_TOKEN"`

### Log soubory rostou příliš rychle
```bash
# Automatické čištění starých logů
find .monitoring/ -name "*.log" -mtime +30 -delete
```

## 🔐 Bezpečnostní poznámky

### Personal Access Token (doporučeno)
Pro vyšší rate limit vytvořte GitHub token:

1. GitHub → Settings → Developer settings → Personal access tokens
2. Generate new token (classic)
3. Scope: pouze "public_repo" (read-only)
4. Přidejte do script jako proměnnou prostředí

```bash
export GITHUB_TOKEN="your_token_here"
# Pak v scriptu použijte: -H "Authorization: token $GITHUB_TOKEN"
```

### Ochrana citlivých informací
- Monitoring script NIKDY neobsahuje hesla nebo API klíče
- Všechny requesty jsou read-only
- Logy neobsahují citlivé informace

## 📈 Monitoring výkonu

### Časové nároky
- Základní monitoring: ~2-5 minut
- Detailní monitoring: ~10-15 minut
- Závisí na počtu dotazů a rychlosti sítě

### Síťové požadavky
- ~5-20 HTTP requestů denně
- Minimální datová náročnost
- Respektuje GitHub rate limity

## 🚨 Akční plán při nalezení kopie

1. **Dokumentujte nález**
   ```bash
   # Vytvořte screenshot a uložte všechny důkazy
   curl -s "URL_OF_COPIED_REPO" > evidence/copied_repo_$(date +%Y%m%d).html
   ```

2. **Analyzujte rozsah kopírování**
   - Kolik kódu bylo zkopírováno?
   - Je zachováno autorství?
   - Je dodržena licence?

3. **Kontaktujte porušovatele**
   - Nejprve přátelský kontakt
   - Požádejte o přidání správného copyright
   - Nebo o odstranění kódu

4. **Eskalace**
   - DMCA takedown notice přes GitHub
   - Právní konzultace pro komerční případy
   - Dokumentace pro budoucí patent filing

## 📞 Podpora

Pro otázky nebo problémy:
- **Email:** petrstepanek99@gmail.com
- **Subject:** "SQL Graph Visualizer - Monitoring Support"

---

**Copyright (c) 2025 Petr Miroslav Stepanek**  
*Součást SQL Graph Visualizer - Dual License*
