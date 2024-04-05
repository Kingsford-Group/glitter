\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{r01preamble}[2018/10/19 R01 Preamble LaTeX class]
\LoadClass[11pt,oneside]{article}

% This documentclass implements the default formatting and typesetting
% environment for `glitter weave`

\RequirePackage[utf8]{inputenc}
\RequirePackage[T1]{fontenc}
\RequirePackage[letterpaper,centering,margin=1in]{geometry}
\RequirePackage{microtype}
\RequirePackage{secdot}
\RequirePackage{parskip}
\RequirePackage{xspace}
\RequirePackage{amssymb}
\RequirePackage{amsmath}
\RequirePackage{amsthm}
\RequirePackage{amsfonts}
\RequirePackage{booktabs}
\RequirePackage[dvipsnames]{xcolor}
\RequirePackage{mathptmx}
\RequirePackage[mode=match, group-separator={,}, tight-spacing=true]{siunitx}
\RequirePackage{url}
\RequirePackage{tikz}
\RequirePackage{listings}

\DeclareOption*{\PassOptionsToClass{\CurrentOption}{article}}
\ProcessOptions\relax

\urlstyle{rm}
\renewcommand{\ttdefault}{pcr}

\lstset{%
    language=Go,%
    basicstyle=\ttfamily\footnotesize,%
    texcl=true,%
    escapechar=\#,%
    extendedchars=true,%
    xleftmargin=0em,%
    showlines=false,
    numbers=left,
}

\widowpenalty=0
\clubpenalty=0

%% These commands are ignored by LaTeX, but read by glitter to defined
%% how various glitter components are typeset.

%%glitter Start         \documentclass{glittertex}
%%glitter StartBook     \glitterStartBook
%%glitter EndBook       \glitterEndBook
%%glitter StartText     \glitterStartText
%%glitter EndText       \glitterEndText$n
%%glitter StartCode     \glitterStartCode{$1}$n\begin{lstlisting}
%%glitter EndCode       \end{lstlisting}\glitterEndCode$n
%%glitter CodeEscape    #
%%glitter CodeCodeRef   #\glitterCodeRef{$1}#
%%glitter TextCodeRef   \glitterCodeRef{$1}
%%glitter CodeEscapeSub #\glitterHash#

%%glitter WeaveLineRef  %%line $1 "$2"
%%glitter TangleLineRef //line $1 "$2"
%%glitter WeaveCommand  pdflatex $1
%%glitter TangleCommand go build

\newcommand\glitterStartBook{\begin{document}}
\newcommand\glitterEndBook{\end{document}}
\newcommand\glitterStartText{\par}
\newcommand\glitterEndText{\par}
\newcommand\glitterCodeRef[1]{\ensuremath{\langle\kern-2pt\langle\textrm{#1}\rangle\kern-2pt\rangle}}
\newcommand\glitterStartCode[1]{\par\noindent\hbox{$\glitterCodeRef{#1}\equiv$}}
\newcommand\glitterEndCode{}
\newcommand\glitterHash{\texttt{\char35}}
