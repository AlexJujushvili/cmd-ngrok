@echo off
set /p msg="შეიყვანეთ commit message: "

:: პროცესის დაწყება
git add .
git commit -m "%msg%"
git push

echo.
echo ცვლილებები წარმატებით აიტვირთა!
pause